// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: device operation lib
// Author: zhangwei
// Create: 2018-01-18

package libdevice

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	hconfig "isula.org/syscontainer-tools/config"
	"isula.org/syscontainer-tools/container"
	"isula.org/syscontainer-tools/libdevice/nsexec"
	"isula.org/syscontainer-tools/pkg/udevd"
	"isula.org/syscontainer-tools/types"
	"isula.org/syscontainer-tools/utils"
)

func checkDevice(config hconfig.ContainerConfig, devs []*types.Device, opts *types.AddDeviceOptions) error {
	devices := devs
	// check all devices, config updated
	if len(devs) == 0 {
		for _, dm := range config.GetAllDevices() {
			devices = append(devices, &types.Device{
				Type:       dm.Type,
				Major:      dm.Major,
				Minor:      dm.Minor,
				PathOnHost: dm.PathOnHost,
				Path:       dm.PathInContainer,
			})
		}
	}

	checkDeviceQos := func(qos *types.Qos) bool {
		for _, device := range devices {
			if device.Major == qos.Major && device.Minor == qos.Minor {
				return true
			}
		}
		return false
	}
	qosOpts := append(opts.ReadBps, opts.WriteBps...)
	qosOpts = append(qosOpts, opts.ReadIOPS...)
	qosOpts = append(qosOpts, opts.WriteIOPS...)
	for _, opt := range qosOpts {
		if !checkDeviceQos(opt) {
			return fmt.Errorf("device %v was not added to container or not in add-device args", opt.Path)
		}
	}
	return nil
}

// UpdateDeviceOwner update device owner
func UpdateDeviceOwner(spec *specs.Spec, device *types.Device) {
	if spec == nil {
		return
	}

	uid, gid := utils.GetUIDGid(spec)

	if uid != -1 {
		device.UID = uint32(uid)
	}
	if gid != -1 {
		device.GID = uint32(gid)
	}
}

// AddDevice will add devices to a container.
func AddDevice(c *container.Container, devices []*types.Device, opts *types.AddDeviceOptions) error {
	driver := nsexec.NewDefaultNsDriver()
	pid := strconv.Itoa(c.Pid())

	innerPath, err := c.GetCgroupPath()
	if err != nil {
		return err
	}

	cgroupPath, err := FindCgroupPath(pid, "devices", innerPath)
	if err != nil {
		return err
	}

	for _, device := range devices {
		UpdateDeviceOwner(c.GetSpec(), device)
	}

	udevdCtrl := udevd.NewUdevdController()

	// lockFile := <container config path>/lock
	// 1. use file lock, to make sure only one process to access this config file.
	// 2. different container has different lock, will not block other container.
	if err := c.Lock(); err != nil {
		return err
	}
	defer c.Unlock()

	// create config file handler.
	config, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return err
	}

	if err := checkDevice(config, devices, opts); err != nil {
		return err
	}

	defer func() {
		if err := config.Flush(); err != nil {
			logrus.Infof("config Flush error:%v", err)
		}
	}()

	if err := udevdCtrl.Lock(); err != nil {
		return err
	}
	defer udevdCtrl.Unlock()

	if err := udevdCtrl.LoadRules(); err != nil {
		return err
	}
	defer udevdCtrl.ToDisk()

	var retErr []error
	// add device and udpate cgroup here
	for _, device := range devices {
		// update config here
		if err = config.UpdateDevice(device, true); err != nil {
			retErr = append(retErr, err)
			continue
		}

		r := &udevd.Rule{
			Name:       device.PathOnHost,
			Container:  c.ContainerID(),
			CtrDevName: device.Path,
		}
		if device.Type != "c" {
			devType, err := types.GetDeviceType(device.PathOnHost)
			if err != nil {
				retErr = append(retErr, err)
				config.UpdateDevice(device, false)
				continue
			}
			if devType == "disk" {
				udevdCtrl.AddRule(r)
			}
		}
		// Do not insert device and update cgroup when:
		// 1. update-config-only flag is set
		// 2. container isn't running (pid==0)
		if !opts.UpdateConfigOnly && c.Pid() > 0 && c.CheckPidExist() {
			// add device to container.
			if err = driver.AddDevice(pid, device, opts.Force); err != nil {
				retErr = append(retErr, err)
				// roll back config and udev rules
				config.UpdateDevice(device, false)
				udevdCtrl.RemoveRule(r)
				continue
			}
			// update cgroup access permission.
			if err = UpdateCgroupPermission(cgroupPath, device, true); err != nil {
				retErr = append(retErr, err)
				// roll back config and udev rules and remove device
				driver.RemoveDevice(pid, device)
				config.UpdateDevice(device, false)
				udevdCtrl.RemoveRule(r)
				continue
			}
		}

		fmt.Fprintf(os.Stdout, "Add device (%s) to container(%s,%s) done.\n", device.PathOnHost, c.Name(), device.Path)
		logrus.Infof("Add device (%s) to container(%s,%s) done", device.PathOnHost, c.Name(), device.Path)
	}

	if err := updateQos(config, pid, innerPath, opts); err != nil {
		return err
	}

	if len(retErr) == 0 {
		return nil
	}
	for i := 0; i < len(retErr); i++ {
		retErr[i] = fmt.Errorf("%s", retErr[i].Error())
	}
	return errors.New(strings.Trim(fmt.Sprint(retErr), "[]"))
}

// UpdateDevice will update device for container.
func UpdateDevice(c *container.Container, opts *types.AddDeviceOptions) error {
	pid := strconv.Itoa(c.Pid())

	innerPath, err := c.GetCgroupPath()
	if err != nil {
		return err
	}

	if err := c.Lock(); err != nil {
		return err
	}
	defer c.Unlock()

	// create config file handler.
	config, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return err
	}

	if err := checkDevice(config, []*types.Device{}, opts); err != nil {
		return err
	}

	defer func() {
		if err := config.Flush(); err != nil {
			logrus.Infof("config Flush error:%v", err)
		}
	}()

	if err := updateQos(config, pid, innerPath, opts); err != nil {
		return err
	}

	return nil
}

// RemoveDevice will remove devices from container
func RemoveDevice(c *container.Container, devices []*types.Device, followPartition bool) error {
	driver := nsexec.NewDefaultNsDriver()
	pid := strconv.Itoa(c.Pid())

	innerPath, err := c.GetCgroupPath()
	if err != nil {
		return err
	}

	cgroupPath, err := FindCgroupPath(pid, "devices", innerPath)
	if err != nil {
		return err
	}

	if err := c.Lock(); err != nil {
		return err
	}
	defer c.Unlock()

	config, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return err
	}
	defer config.Flush()

	udevdCtrl := udevd.NewUdevdController()
	if err := udevdCtrl.Lock(); err != nil {
		return err
	}
	defer udevdCtrl.Unlock()

	if err := udevdCtrl.LoadRules(); err != nil {
		return err
	}
	defer udevdCtrl.ToDisk()

	var retErr []error
	var newDevices []*types.Device
	for _, device := range devices {
		newDevice := config.FindDeviceByMapping(device)
		if newDevice == nil {
			errinfo := fmt.Sprint("Device pair(", device.PathOnHost, ":", device.Path, ") is not added by syscontainer-tools, can not remove it, please check input parameter.")
			retErr = append(retErr, errors.New(errinfo))
			continue
		}
		newDevices = append(newDevices, newDevice)

		if followPartition {
			subDevices := config.FindSubPartition(newDevice)
			for _, subDev := range subDevices {
				// check the sub partition is added by syscontainer-tools
				if found := config.FindDeviceByMapping(subDev); found == nil {
					continue
				}
				found := false
				for _, eDev := range newDevices {
					if subDev.Path == eDev.Path && subDev.PathOnHost == eDev.PathOnHost {
						found = true
						break
					}
				}
				if !found {
					newDevices = append(newDevices, subDev)
				}
			}
		}
	}

	for _, device := range newDevices {
		// update config.
		if err = config.UpdateDevice(device, false); err != nil {
			retErr = append(retErr, err)
			continue
		}
		r := &udevd.Rule{
			Name:       device.PathOnHost,
			CtrDevName: device.Path,
			Container:  c.ContainerID(),
		}
		udevdCtrl.RemoveRule(r)

		// only update for running container
		if c.Pid() > 0 && c.CheckPidExist() {
			if err = driver.RemoveDevice(pid, device); err != nil {
				config.UpdateDevice(device, true)
				if device.Type != "c" {
					devType, err := types.GetDeviceType(device.PathOnHost)
					if err != nil {
						retErr = append(retErr, err)
						config.UpdateDevice(device, true)
						continue
					}
					if devType == "disk" {
						udevdCtrl.AddRule(r)
					}
				}
				retErr = append(retErr, err)
				continue
			}

			// update cgroup access permission.
			if err = UpdateCgroupPermission(cgroupPath, device, false); err != nil {
				// TODO: also need a roll back?
				retErr = append(retErr, err)
				continue
			}
		}

		fmt.Fprintf(os.Stdout, "Remove device (%s) from container(%s,%s) done.\n", device.PathOnHost, c.Name(), device.Path)
		logrus.Infof("Remove device (%s) from container(%s,%s) done.\n", device.PathOnHost, c.Name(), device.Path)

		if err := removeQos(config, pid, innerPath, device); err != nil {
			retErr = append(retErr, err)
		}
	}

	if len(retErr) == 0 {
		return nil
	}
	for i := 0; i < len(retErr); i++ {
		retErr[i] = fmt.Errorf("%s", retErr[i].Error())
	}
	return errors.New(strings.Trim(fmt.Sprint(retErr), "[]"))
}

// ListDevice list container devices
func ListDevice(c *container.Container) ([]*hconfig.DeviceMapping, []*hconfig.DeviceMapping, error) {
	if err := c.Lock(); err != nil {
		return nil, nil, err
	}
	defer c.Unlock()
	hConfig, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return nil, nil, err
	}

	allDevice := hConfig.GetAllDevices()
	var majorDevices []*hconfig.DeviceMapping

	for _, sDevice := range allDevice {
		if sDevice.Parent == "" {
			majorDevices = append(majorDevices, sDevice)
		}
	}

	return allDevice, majorDevices, nil
}

// AddPath will add paths from host to container
func AddPath(c *container.Container, binds []*types.Bind) error {
	driver := nsexec.NewDefaultNsDriver()
	pid := strconv.Itoa(c.Pid())

	if err := c.Lock(); err != nil {
		return fmt.Errorf("AddPath: failed to get lock, err: %s", err)
	}
	defer c.Unlock()

	config, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return fmt.Errorf("AddPath: failed to create config, err: %s", err)
	}
	defer config.Flush()

	if err := config.CheckPathNum(); err != nil {
		return err
	}

	var retErr []error
	for _, bind := range binds {
		logrus.Debugf("Adding path: %+v", bind)
		// 1. update config.
		hostPathExist, err := config.UpdateBind(bind, true)
		if err != nil {
			return fmt.Errorf("AddPath: failed to UpdateBind, err: %s", err)
		}

		if c.Pid() > 0 && c.CheckPidExist() {
			// 2. prepare transferpath if needed
			if err := utils.PrepareTransferPath("/", c.ContainerID(), bind, !hostPathExist); err != nil {
				config.UpdateBind(bind, false)

				// if no existed, do unmount
				if !hostPathExist {
					utils.RemoveTransferPath(c.ContainerID(), bind)
				}
				return fmt.Errorf("AddPath: failed to prepare transfer base, err: %s", err)
			}
			if err = driver.AddBind(pid, bind); err != nil {
				retErr = append(retErr, err)
				config.UpdateBind(bind, false)
				if !hostPathExist {
					utils.RemoveTransferPath(c.ContainerID(), bind)
				}
				return fmt.Errorf("AddPath: failed to add bind, err: %s", err)
			}
		}
		msg := fmt.Sprintf("Add path (%s) to container(%s,%s) done.", bind.HostPath, c.Name(), bind.ContainerPath)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	if len(retErr) == 0 {
		return nil
	}
	for i := 0; i < len(retErr); i++ {
		retErr[i] = fmt.Errorf("%s", retErr[i].Error())
	}
	return errors.New(strings.Trim(fmt.Sprint(retErr), "[]"))
}

// RemovePath will remove paths from container
func RemovePath(c *container.Container, binds []*types.Bind) error {

	driver := nsexec.NewDefaultNsDriver()
	pid := strconv.Itoa(c.Pid())
	if err := c.Lock(); err != nil {
		return err
	}
	defer c.Unlock()

	config, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return err
	}
	defer config.Flush()

	var retErr []error
	for _, bind := range binds {
		mp, err := config.GetBindInConfig(bind)
		if err != nil {
			retErr = append(retErr, err)
			continue
		}

		if mp == nil {
			errinfo := fmt.Sprint("Path pair(", bind.HostPath, ":", bind.ContainerPath, ") is not added by syscontainer-tools, can not remove it, please check input parameter")
			retErr = append(retErr, errors.New(errinfo))
			continue
		}
		bind.MountOption = mp.Permission

		// update config.
		removeHostPath, err := config.UpdateBind(bind, false)
		if err != nil {
			retErr = append(retErr, fmt.Errorf("Failed to update bind(%v), error: %s, still try to remove it", bind, err))
		}

		// remove from container.
		if c.Pid() > 0 && c.CheckPidExist() {
			if err := driver.RemoveBind(pid, bind); err != nil {
				retErr = append(retErr, fmt.Errorf("Failed to remove bind(%v),error: %s", bind, err))
				config.UpdateBind(bind, true)
				continue
			}
			if removeHostPath == true {
				if err := utils.RemoveTransferPath(c.ContainerID(), bind); err != nil {
					retErr = append(retErr, fmt.Errorf("Remove path (%s) from %s failed, err: %s", bind.HostPath, c.Name(), err))

				}
			}
		}
		msg := fmt.Sprintf("Remove path (%s) from container(%s,%s) done", bind.HostPath, c.Name(), bind.ContainerPath)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}

	if len(retErr) == 0 {
		return nil
	}
	for i := 0; i < len(retErr); i++ {
		retErr[i] = fmt.Errorf("%s", retErr[i].Error())
	}
	return errors.New(strings.Trim(fmt.Sprint(retErr), "[]"))
}

// ListPath list container paths
func ListPath(ctr *container.Container) ([]string, error) {
	if err := ctr.Lock(); err != nil {
		return nil, err
	}
	defer ctr.Unlock()
	hConfig, err := hconfig.NewContainerConfig(ctr)
	if err != nil {
		return nil, err
	}

	return hConfig.GetBinds(), nil
}

func updateQos(config hconfig.ContainerConfig, pid, innerPath string, opts *types.AddDeviceOptions) error {
	// update device read iops
	for _, devReadIOPS := range opts.ReadIOPS {
		if err := UpdateCgroupDeviceReadIOPS(pid, innerPath, devReadIOPS.String()); err != nil {
			return err
		}
		if err := config.UpdateDeviceQos(devReadIOPS, hconfig.QosReadIOPS); err != nil {
			return err
		}
		msg := fmt.Sprintf("Update read iops for device (%s,%s) done.", devReadIOPS.Path, devReadIOPS.Value)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	// update device write iops
	for _, devWriteIOPS := range opts.WriteIOPS {
		if err := UpdateCgroupDeviceWriteIOPS(pid, innerPath, devWriteIOPS.String()); err != nil {
			return err
		}
		if err := config.UpdateDeviceQos(devWriteIOPS, hconfig.QosWriteIOPS); err != nil {
			return err
		}
		msg := fmt.Sprintf("Update write iops for device (%s,%s) done.", devWriteIOPS.Path, devWriteIOPS.Value)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)
	}
	// update device read bps
	for _, devReadBps := range opts.ReadBps {
		if err := UpdateCgroupDeviceReadBps(pid, innerPath, devReadBps.String()); err != nil {
			return err
		}
		if err := config.UpdateDeviceQos(devReadBps, hconfig.QosReadBps); err != nil {
			return err
		}
		msg := fmt.Sprintf("Update read bps for device (%s,%s) done.", devReadBps.Path, devReadBps.Value)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	// update device write bps
	for _, devWriteBps := range opts.WriteBps {
		if err := UpdateCgroupDeviceWriteBps(pid, innerPath, devWriteBps.String()); err != nil {
			return err
		}
		if err := config.UpdateDeviceQos(devWriteBps, hconfig.QosWriteBps); err != nil {
			return err
		}
		msg := fmt.Sprintf("Update write bps for device (%s,%s) done.", devWriteBps.Path, devWriteBps.Value)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	// update device blkio weight
	for _, devBlkioWeight := range opts.BlkioWeight {
		cfqEnable, err := devBlkioWeight.GetCfqAbility()
		if err == nil && cfqEnable {
			if err := UpdateCgroupDeviceWeight(pid, innerPath, devBlkioWeight.String()); err != nil {
				return err
			}
			if err := config.UpdateDeviceQos(devBlkioWeight, hconfig.QosBlkioWeight); err != nil {
				return err
			}
			msg := fmt.Sprintf("Update blkio weight for device (%s,%s) done.", devBlkioWeight.Path, devBlkioWeight.Value)
			fmt.Fprintln(os.Stdout, msg)
			logrus.Info(msg)

		} else {
			msg := fmt.Sprintf("device not support cfq:%s", devBlkioWeight.Path)
			fmt.Fprintln(os.Stdout, msg)
			logrus.Info(msg)

		}
	}
	return nil
}

func removeQos(config hconfig.ContainerConfig, pid, innerPath string, device *types.Device) error {
	cleanString := fmt.Sprintf("%d:%d 0", device.Major, device.Minor)
	if exist, err := config.RemoveDeviceQos(device, hconfig.QosReadIOPS); err != nil {
		return err
	} else if exist {
		if err := UpdateCgroupDeviceReadIOPS(pid, innerPath, cleanString); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Remove read iops for device (%s) done.\n", device.PathOnHost)
	}
	if exist, err := config.RemoveDeviceQos(device, hconfig.QosWriteIOPS); err != nil {
		return err
	} else if exist {
		if err := UpdateCgroupDeviceWriteIOPS(pid, innerPath, cleanString); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Remove write iops for device (%s) done.\n", device.PathOnHost)
	}
	if exist, err := config.RemoveDeviceQos(device, hconfig.QosReadBps); err != nil {
		return err
	} else if exist {
		if err := UpdateCgroupDeviceReadBps(pid, innerPath, cleanString); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Remove read bps for device (%s) done.\n", device.PathOnHost)
	}
	if exist, err := config.RemoveDeviceQos(device, hconfig.QosWriteBps); err != nil {
		return err
	} else if exist {
		if err := UpdateCgroupDeviceWriteBps(pid, innerPath, cleanString); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Remove write bps for device (%s) done.\n", device.PathOnHost)
	}
	if exist, err := config.RemoveDeviceQos(device, hconfig.QosBlkioWeight); err != nil {
		return err
	} else if exist {
		if err := UpdateCgroupDeviceWeight(pid, innerPath, cleanString); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Remove blkio weight for device (%s) done.\n", device.PathOnHost)
	}
	return nil
}
