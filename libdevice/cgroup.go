// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: cgroup operation for container
// Author: zhangwei
// Create: 2018-01-18

package libdevice

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"isula.org/isulad-tools/types"
)

var (
	cgroupNamePrefix = "name="
)

// GetCgroupDir returns the cgorup mount directory from pid
func GetCgroupDir(pid, subsystem string) (string, error) {
	path := filepath.Join("/proc", pid, "cgroup")
	cgroupmap, err := cgroups.ParseCgroupFile(path)
	if err != nil {
		return "", err
	}

	if path, ok := cgroupmap[subsystem]; ok {
		return path, nil
	}
	if path, ok := cgroupmap[cgroupNamePrefix+subsystem]; ok {
		return path, nil
	}
	return "", fmt.Errorf("Error: ControllerPath of %s is not found", subsystem)
}

// FindCgroupPath will search the subsystem cgroup path for target process
func FindCgroupPath(pid, subsystem, innerPath string) (string, error) {
	cgroupRoot, err := cgroups.FindCgroupMountpointDir()
	if err != nil {
		return "", err
	}

	mnt, root, err := cgroups.FindCgroupMountpointAndRoot(subsystem)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(innerPath) {
		return filepath.Join(cgroupRoot, filepath.Base(mnt), innerPath), nil
	}
	initPath, err := GetCgroupDir(pid, subsystem)
	if err != nil {
		return "", err
	}

	// This is needed for nested containers, because in /proc/pid/cgroup we
	// see pathes from host, which don't exist in container.
	relDir, err := filepath.Rel(root, initPath)
	if err != nil {
		return "", err
	}

	return filepath.Join(mnt, relDir), nil
}

// UpdateCgroupPermission will update the cgroup permissions for specified device
func UpdateCgroupPermission(CgroupBase string, device *types.Device, isAddDevice bool) error {
	var path string

	if isAddDevice {
		path = filepath.Join(CgroupBase, "devices.allow")
	} else {
		path = filepath.Join(CgroupBase, "devices.deny")
	}
	value := device.CgroupString()
	if err := ioutil.WriteFile(path, []byte(value), 0600); err != nil {
		return err
	}

	return nil
}

// UpdateCgroupDeviceReadIOPS updates the read device iops for container/pid
func UpdateCgroupDeviceReadIOPS(pid, innerPath, value string) error {
	if pid == "0" {
		return nil
	}

	cgroupPath, err := FindCgroupPath(pid, "blkio", innerPath)
	if err != nil {
		return err
	}
	path := filepath.Join(cgroupPath, "blkio.throttle.read_iops_device")
	if err := ioutil.WriteFile(path, []byte(value), 0600); err != nil {
		return err
	}
	return nil
}

// UpdateCgroupDeviceWriteIOPS updates the write device iops for container/pid
func UpdateCgroupDeviceWriteIOPS(pid, innerPath, value string) error {
	if pid == "0" {
		return nil
	}

	cgroupPath, err := FindCgroupPath(pid, "blkio", innerPath)
	if err != nil {
		return err
	}
	path := filepath.Join(cgroupPath, "blkio.throttle.write_iops_device")
	if err := ioutil.WriteFile(path, []byte(value), 0600); err != nil {
		return err
	}
	return nil
}

// UpdateCgroupDeviceReadBps updates the read device bps for container/pid
func UpdateCgroupDeviceReadBps(pid, innerPath, value string) error {
	if pid == "0" {
		return nil
	}

	cgroupPath, err := FindCgroupPath(pid, "blkio", innerPath)
	if err != nil {
		return err
	}
	path := filepath.Join(cgroupPath, "blkio.throttle.read_bps_device")
	if err := ioutil.WriteFile(path, []byte(value), 0600); err != nil {
		return err
	}
	return nil
}

// UpdateCgroupDeviceWriteBps updates the write device bps for container/pid
func UpdateCgroupDeviceWriteBps(pid, innerPath, value string) error {
	if pid == "0" {
		return nil
	}

	cgroupPath, err := FindCgroupPath(pid, "blkio", innerPath)
	if err != nil {
		return err
	}
	path := filepath.Join(cgroupPath, "blkio.throttle.write_bps_device")
	if err := ioutil.WriteFile(path, []byte(value), 0600); err != nil {
		return err
	}
	return nil
}

// UpdateCgroupDeviceWeight updates the write device weight for container/pid
func UpdateCgroupDeviceWeight(pid, innerPath, value string) error {
	if pid == "0" {
		return nil
	}

	cgroupPath, err := FindCgroupPath(pid, "blkio", innerPath)
	if err != nil {
		return err
	}
	path := filepath.Join(cgroupPath, "blkio.weight_device")
	if err := ioutil.WriteFile(path, []byte(value), 0600); err != nil {
		return fmt.Errorf("%s, please check whether current OS support blkio weight device configuration for bfq scheduler", err)
	}
	return nil
}
