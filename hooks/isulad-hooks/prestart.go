// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: prestart hook
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/configs"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	hconfig "isula.org/isulad-tools/config"
	"isula.org/isulad-tools/libdevice"
	"isula.org/isulad-tools/libdevice/nsexec"
	"isula.org/isulad-tools/libnetwork"
	"isula.org/isulad-tools/pkg/udevd"
	"isula.org/isulad-tools/types"
	"isula.org/isulad-tools/utils"
)

const (
	arrayLen = 3 // calc path for bind array len
	minor    = 7 // runcDevice Minor
	major    = 5 // runcDevice Major
)

// HookAction is the definition of hook action callback
type HookAction func(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error

func fmPtr(mode int64) *os.FileMode {
	fm := os.FileMode(mode)
	return &fm
}

// re-parse device
func calcPathForDevice(rootfs string, device *hconfig.DeviceMapping) string {
	paths := []string{device.PathOnHost, filepath.Join(rootfs, device.PathInContainer), device.CgroupPermissions}
	return strings.Join(paths, ":")
}

func calcPathForBind(rootfs string, bind string) (string, error) {
	array := strings.SplitN(bind, ":", 3)
	if len(array) < arrayLen {
		// this should not happen.
		// if happen, parseBind will print error
		return "", fmt.Errorf("Error: bind lack of \":\"")
	}
	paths := []string{array[0], filepath.Join(rootfs, array[1]), array[2]}
	return strings.Join(paths, ":"), nil
}

// AddDevices will add devices to the container
func AddDevices(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	pid := strconv.Itoa(state.Pid)
	driver := nsexec.NewDefaultNsDriver()

	cgroupPath, err := libdevice.FindCgroupPath(pid, "devices", spec.Linux.CgroupsPath)
	if err != nil {
		return err
	}
	udevdCtrl := udevd.NewUdevdController()
	if err := udevdCtrl.Lock(); err != nil {
		return err
	}
	defer udevdCtrl.Unlock()

	if err := udevdCtrl.LoadRules(); err != nil {
		return err
	}
	defer func() {
		logrus.Infof("Start sync rules to disk")
		udevdCtrl.ToDisk()
		logrus.Infof("Finish sync rules to disk")
	}()

	for index, dev := range hookConfig.Devices {
		// re-calc the dest path of device.
		resolvDev := calcPathForDevice(state.Root, dev)
		device, err := libdevice.ParseDevice(resolvDev)
		if err != nil {
			logrus.Errorf("[device-hook] Add device (%s), parse device failed: %v", resolvDev, err)
			return err
		}

		// update config here
		if dev.Major != device.Major || dev.Minor != device.Minor {
			hookConfig.Devices[index].Major = device.Major
			hookConfig.Devices[index].Minor = device.Minor
			hookConfig.SetConfigDirty()
		}

		if device.Type != "c" {
			devType, err := types.GetDeviceType(device.PathOnHost)
			if err != nil {
				return err
			}
			if devType == "disk" {
				udevdCtrl.AddRule(&udevd.Rule{
					Name:       dev.PathOnHost,
					CtrDevName: dev.PathInContainer,
					Container:  state.ID,
				})
			}
		}

		// use exec driver to add device.
		libdevice.UpdateDeviceOwner(spec, device)
		if err = driver.AddDevice(pid, device, true); err != nil {
			logrus.Errorf("[device-hook] Add device (%s) failed: %v", resolvDev, err)
			return err
		}

		// update cgroup access permission.
		if err = libdevice.UpdateCgroupPermission(cgroupPath, device, true); err != nil {
			logrus.Errorf("[device-hook] Update add device (%s) cgroup failed: %v", resolvDev, err)
			return err
		}
	}
	return nil
}

// AddBinds will add the binds to the container
func AddBinds(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	pid := strconv.Itoa(state.Pid)
	driver := nsexec.NewDefaultNsDriver()

	for _, bindstr := range hookConfig.Binds {
		bind, err := stringToBind(state.Root, bindstr, spec, true)
		if err != nil {
			logrus.Errorf("[device-hook] parse bind error, %s, skipping", err)
			continue
		}
		// re-calc the bind dest path, because we have not done the chroot
		if err := utils.PrepareTransferPath(state.Root, state.ID, bind, true); err != nil {
			logrus.Errorf("[device-hook] Prepare tansfer path (%s) failed, prepare tansfer path failed: %v", bindstr, err)
		}

		if err = driver.AddBind(pid, bind); err != nil {
			logrus.Errorf("[device-hook] Add bind (%s) failed: %v", bindstr, err)
			continue
		}
	}
	return nil
}

// SharePath will add the binds to the container
func SharePath(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	pid := strconv.Itoa(state.Pid)
	driver := nsexec.NewDefaultNsDriver()
	if err := utils.PrepareHostPath(state.ID); err != nil {
		return err
	}
	bind := &types.Bind{
		HostPath:      utils.GetContainerSpecDir(state.ID),
		IsDir:         true,
		ContainerPath: filepath.Join(state.Root, utils.GetSlavePath()),
	}
	if err := driver.AddTransferBase(pid, bind); err != nil {
		return err
	}
	return nil

}

// UpdateQos will update the Qos config for container
func UpdateQos(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	pid := strconv.Itoa(state.Pid)
	innerPath := spec.Linux.CgroupsPath

	// update device read iops
	for _, devReadIOPS := range hookConfig.ReadIOPS {
		if err := updateQosDeviceNum(hookConfig, devReadIOPS); err != nil {
			return err
		}

		if err := libdevice.UpdateCgroupDeviceReadIOPS(pid, innerPath, devReadIOPS.String()); err != nil {
			logrus.Errorf("[device-hook] Failed to update device read iops (%s) for container %s: %v", devReadIOPS.String(), state.ID, err)
			return err
		}
	}
	// update device write iops
	for _, devWriteIOPS := range hookConfig.WriteIOPS {
		if err := updateQosDeviceNum(hookConfig, devWriteIOPS); err != nil {
			return err
		}

		if err := libdevice.UpdateCgroupDeviceWriteIOPS(pid, innerPath, devWriteIOPS.String()); err != nil {
			logrus.Errorf("[device-hook] Failed to update device write iops (%s) for container %s: %v", devWriteIOPS, state.ID, err)
			return err
		}
	}
	// update device read bps
	for _, devReadBps := range hookConfig.ReadBps {
		if err := updateQosDeviceNum(hookConfig, devReadBps); err != nil {
			return err
		}

		if err := libdevice.UpdateCgroupDeviceReadBps(pid, innerPath, devReadBps.String()); err != nil {
			logrus.Errorf("[device-hook] Failed to update device read bps (%s) for container %s: %v", devReadBps.String(), state.ID, err)
			return err
		}
	}
	// update device write bps
	for _, devWriteBps := range hookConfig.WriteBps {
		if err := updateQosDeviceNum(hookConfig, devWriteBps); err != nil {
			return err
		}

		if err := libdevice.UpdateCgroupDeviceWriteBps(pid, innerPath, devWriteBps.String()); err != nil {
			logrus.Errorf("[device-hook] Failed to update device write bps (%s) for container %s: %v", devWriteBps.String(), state.ID, err)
			return err
		}
	}
	// update device blkio weight
	for _, devBlkioWeight := range hookConfig.BlkioWeight {
		if err := updateQosDeviceNum(hookConfig, devBlkioWeight); err != nil {
			return err
		}

		if err := libdevice.UpdateCgroupDeviceWeight(pid, innerPath, devBlkioWeight.String()); err != nil {
			logrus.Errorf("[device-hook] Failed to update device weight %s for container %s : %v", devBlkioWeight.String(), state.ID, err)
			return err
		}
	}
	return nil
}

// UpdateNetwork will update the network interface for container
func UpdateNetwork(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	nsPath := fmt.Sprintf("/proc/%d/ns/net", state.Pid)
	if err := os.MkdirAll(hconfig.IsuladToolsDirNetns, 0600); err != nil {
		logrus.Errorf("[device-hook] Failed to Create netns dir %v", err)
		return err
	}
	file, err := os.Create(filepath.Join(hconfig.IsuladToolsDirNetns, state.ID))
	if err != nil {
		logrus.Errorf("[device-hook] Failed to Create netns file %v", err)
		return err
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return err
	}

	if err := unix.Mount(nsPath, file.Name(), "bind", unix.MS_BIND, ""); err != nil {
		logrus.Errorf("[device-hook] Failed to Mount netns file %v", err)
		return err
	}

	for _, nic := range hookConfig.NetworkInterfaces {
		if err := libnetwork.AddNicToContainer(nsPath, nic); err != nil {
			logrus.Errorf("[device-hook] Failed to add network interface (%s) to container %s: %v", nic.String(), state.ID, err)
			return err
		}
	}

	for _, route := range hookConfig.NetworkRoutes {
		if err := libnetwork.AddRouteToContainer(nsPath, route); err != nil {
			logrus.Errorf("[device-hook] Failed to add route rule (%s) to container %s: %v", route.String(), state.ID, err)
			return err
		}
	}
	return nil
}

// DynLoadModule dynamic load kernel modules
func DynLoadModule(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	kernelModules := []string{"-a"}

	for _, env := range spec.Process.Env {
		if strings.Contains(env, "KERNEL_MODULES=") {
			envValue := strings.Split(env, "=")

			if envValue[0] == "KERNEL_MODULES" {
				envModules := strings.Split(envValue[1], ",")

				for _, module := range envModules {
					if module != "" {
						if isValidModuleName(module) {
							kernelModules = append(kernelModules, module)
						} else {
							return fmt.Errorf("[module-hook] Failed to modprobe modules by module name is incorrect:%s", module)
						}
					}
				}
				break
			}
		}
	}

	if len(kernelModules) == 1 {
		return nil
	}

	cmd := exec.Command("modprobe", kernelModules...)

	err := cmd.Run()
	if err != nil {
		logrus.Errorf("[module-hook] Failed to modprobe modules (%q) to host : %v", kernelModules, err)
		return err
	}
	return nil
}

func isValidModuleName(input string) bool {
	pattern := `^[A-Za-z0-9\-\_]*$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(input)
}

// AdjustUserns ajust user namespace for container hooks
func AdjustUserns(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	driver := nsexec.NewDefaultNsDriver()
	pid := strconv.Itoa(state.Pid)

	if len(spec.Linux.UIDMappings) == 0 && len(spec.Linux.GIDMappings) == 0 {
		return nil
	}

	for key, value := range spec.Linux.Sysctl {
		if err := driver.UpdateSysctl(pid, &types.Sysctl{key, value}); err != nil {
			logrus.Errorf("[device-hook] Update sysctl %s:%s failed: %v", key, value, err)
			return err
		}
	}

	containerStoragePath, err := utils.GetContainerStoragePath()
	if err != nil {
		return err
	}
	if strings.Contains(containerStoragePath, "isulad") {
		return nil
	}

	for _, mount := range spec.Mounts {
		if mount.Destination == "/dev" && mount.Type == "tmpfs" && mount.Source == "tmpfs" {
			// remount to allow dev
			rootfsDev := filepath.Join(state.Root, "/dev")
			rootfsBakDev := filepath.Join(state.Root, "/.dev")
			// move /dev to /.dev
			if err := driver.Mount(pid, &types.Mount{Source: rootfsDev, Destination: rootfsBakDev, Type: "move", Options: ""}); err != nil {
				logrus.Errorf("[device-hook] Move /dev to /.dev failed: %v", err)
			}
			// remount /dev whit dev option
			options := append(mount.Options, "dev")
			uid, gid := utils.GetUIDGid(spec)
			if uid != -1 {
				uidStr := fmt.Sprintf("uid=%d", uid)
				options = append(options, uidStr)
			}
			if gid != -1 {
				gidStr := fmt.Sprintf("gid=%d", gid)
				options = append(options, gidStr)
			}
			opts := strings.Join(options, ",")
			if err := driver.Mount(pid, &types.Mount{Source: mount.Type, Destination: rootfsDev, Type: mount.Type, Options: opts}); err != nil {
				logrus.Errorf("[device-hook] mount /dev failed: %v", err)
			}
			options = append(mount.Options, "remount")
			opts = strings.Join(options, ",")
			if err := driver.Mount(pid, &types.Mount{Source: rootfsDev, Destination: rootfsDev, Type: mount.Type, Options: opts}); err != nil {
				logrus.Errorf("[device-hook] remount /dev failed: %v", err)
			}
			if uid == -1 {
				uid = 0
			}
			if gid == -1 {
				gid = 0
			}
			// bind mountpoints in /dev
			for _, mnt := range spec.Mounts {
				if mnt.Destination != "/dev" && strings.Contains(mnt.Destination, "/dev") {
					source := filepath.Join(rootfsBakDev, filepath.Base(mnt.Destination))
					dest := filepath.Join(rootfsDev, filepath.Base(mnt.Destination))
					if err := driver.Mount(pid, &types.Mount{source, dest, "bind", "bind", uid, gid}); err != nil {
						logrus.Errorf("[device-hook] bindmount %s failed: %v", mnt.Destination, err)
					}
				}
			}

			// re-add runc's devices
			runcDevices := []specs.LinuxDevice{
				{
					Type:     "c",
					Path:     "/dev/full",
					Major:    1,
					Minor:    minor,
					FileMode: fmPtr(0666),
				},
				{
					Type:     "c",
					Path:     "/dev/tty",
					Major:    major,
					Minor:    0,
					FileMode: fmPtr(0666),
				},
			}
			spec.Linux.Devices = append(spec.Linux.Devices, runcDevices...)
			for _, mnt := range spec.Linux.Devices {
				if mnt.Path != "/dev" && strings.Contains(mnt.Path, "/dev") {
					source := filepath.Join(rootfsBakDev, filepath.Base(mnt.Path))
					dest := filepath.Join(rootfsDev, filepath.Base(mnt.Path))
					if err := driver.Mount(pid, &types.Mount{source, dest, "bind", "bind", uid, gid}); err != nil {
						logrus.Errorf("[device-hook] bindmount %s failed: %v", mnt.Path, err)
					}
				}
			}
			links := [][2]string{
				{"/proc/self/fd", "/dev/fd"},
				{"/proc/self/fd/0", "/dev/stdin"},
				{"/proc/self/fd/1", "/dev/stdout"},
				{"/proc/self/fd/2", "/dev/stderr"},
				{"pts/ptmx", "/dev/ptmx"},
				{"/proc/kcore", "/dev/core"},
			}
			for _, mnt := range links {
				source := mnt[0]
				dest := filepath.Join(state.Root, mnt[1])
				if err := driver.Mount(pid, &types.Mount{source, dest, "link", "", uid, gid}); err != nil {
					logrus.Errorf("[device-hook] link %s failed: %v", mnt[0], err)
				}
			}
		}
	}

	return nil
}

// prestartHook is the main logic of device hook
func prestartHook(data *hookData, withRelabel bool) error {
	var actions []HookAction
	actions = []HookAction{
		SharePath,
		AdjustUserns,
		AddDevices,
		AddBinds,
		UpdateQos,
		UpdateNetwork,
		DynLoadModule,
	}
	if withRelabel {
		actions = append(actions, PrestartRelabel)
	}
	for _, ac := range actions {
		if err := ac(data.state, data.hookConfig, data.spec); err != nil {
			logrus.Errorf("Failed with err: %v", err)
			return err
		}
	}
	return nil
}

func updateQosDeviceNum(hookConfig *hconfig.ContainerHookConfig, qos *types.Qos) error {
	devMajor, devMinor, err := libdevice.GetDeviceNum(qos.Path)
	if err != nil {
		logrus.Errorf("[device-hook] Failed to update device num (%s) for container %v", qos.String(), err)
		return err
	}

	hookConfig.UpdateQosDevNum(qos, devMajor, devMinor)
	return nil
}
