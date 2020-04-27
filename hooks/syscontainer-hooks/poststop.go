// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: poststop hook
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/configs"
	hconfig "isula.org/syscontainer-tools/config"
	"isula.org/syscontainer-tools/libdevice"
	"isula.org/syscontainer-tools/libnetwork"
	"isula.org/syscontainer-tools/pkg/udevd"
	"isula.org/syscontainer-tools/types"
	"isula.org/syscontainer-tools/utils"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	_ "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// RemoveUdevRule will remove device udev rule for the stopped container
func RemoveUdevRule(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	udevdCtrl := udevd.NewUdevdController()
	if err := udevdCtrl.Lock(); err != nil {
		return err
	}
	defer udevdCtrl.Unlock()

	if err := udevdCtrl.LoadRules(); err != nil {
		return err
	}
	defer udevdCtrl.ToDisk()
	for _, dev := range hookConfig.Devices {
		// re-calc the dest path of device.
		resolvDev := calcPathForDevice(state.Root, dev)
		device, err := libdevice.ParseDevice(resolvDev)
		if err != nil {
			logrus.Errorf("[device-hook] Add device (%s), parse device failed: %v", resolvDev, err)
			continue
		}

		if device.Type == "c" {
			continue
		}

		devType, err := types.GetDeviceType(device.PathOnHost)
		if err != nil {
			return err
		}
		if devType == "disk" {
			udevdCtrl.RemoveRule(&udevd.Rule{
				Name:       dev.PathOnHost,
				CtrDevName: dev.PathInContainer,
				Container:  state.ID,
			})
		}
	}

	return nil
}

// RemoveNetworkDevices will remove network device after container stop.
func RemoveNetworkDevices(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {

	file, err := os.Open(filepath.Join(hconfig.IsuladToolsDirNetns, state.ID))
	if err != nil {
		logrus.Errorf("[device-hook] Failed to Open netns file %v", err)
		return fmt.Errorf("[device-hook] Failed to Open netns file %v", err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			logrus.Errorf("Failed to remove fileName err: %v", err)
		}
		file.Close()
	}()

	for _, nic := range hookConfig.NetworkInterfaces {
		err := libnetwork.DelNicFromContainer(filepath.Join(hconfig.IsuladToolsDirNetns, state.ID), nic)
		if err != nil {
			logrus.Errorf("[device-hook] Failed to del network interface (%s) from container %s: %v", nic.String(), state.ID, err)
			continue
		}
		logrus.Debugf("Removed %s interface: (%s,%s)", nic.Type, nic.HostNicName, nic.CtrNicName)
	}

	if err = unix.Unmount(file.Name(), unix.MNT_DETACH); err != nil {
		err = fmt.Errorf("[device-hook] Failed to Unmount netns file %v", err)
		logrus.Errorf("%v", err)
	}

	return err
}

func stringToBind(containerRoot, bindstr string, spec *specs.Spec, isCreate bool) (*types.Bind, error) {
	// we have done the chroot
	resolvBind, err := calcPathForBind(containerRoot, bindstr)
	if err != nil {
		return nil, fmt.Errorf("Re-Calculate bind(%s) failed: %v", bindstr, err)
	}

	bind, err := libdevice.ParseBind(resolvBind, spec, isCreate)
	if err != nil {
		return nil, fmt.Errorf("Parse bind(%s) failed: %v", bindstr, err)
	}
	return bind, nil

}

// RemoveSharedPath will remove shared path after container stop.
func RemoveSharedPath(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {

	for _, bindstr := range hookConfig.Binds {
		// we have do the chroot
		bind, err := stringToBind("/", bindstr, spec, false)
		if err != nil {
			logrus.Errorf("RemoveSharedPath failed: %s", err)
			continue
		}
		if err := utils.RemoveTransferPath(state.ID, bind); err != nil {
			logrus.Errorf("RemoveSharedPath failed: Path: %v failed: %s", bind, err)
		}

	}
	utils.RemoveContainerSpecPath(state.ID)
	return nil

}

// prestartHook is the main logic of device hook
func postStopHook(data *hookData, withRelabel bool) {
	var actions []HookAction
	actions = []HookAction{RemoveUdevRule, RemoveNetworkDevices, RemoveSharedPath}
	if withRelabel {
		actions = append(actions, PostStopRelabel)
	}
	for _, ac := range actions {
		if err := ac(data.state, data.hookConfig, data.spec); err != nil {
			logrus.Errorf("Failed with err: %v", err)
		}
	}
}
