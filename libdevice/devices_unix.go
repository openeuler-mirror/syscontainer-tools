// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: device operation
// Author: zhangwei
// Create: 2018-01-18

// +build linux freebsd

package libdevice

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"isula.org/syscontainer-tools/types"

	"github.com/sirupsen/logrus"
)

var (
	// ErrNotADevice not a device error
	ErrNotADevice = errors.New("not a device")
)

// Testing dependencies
var (
	osLstat = os.Lstat
)

// ParseMapping will return a device with mapping segment only.
func ParseMapping(device string) (*types.Device, error) {
	var src, dst, permissions string
	arr := strings.Split(device, ":")
	permissions = "rwm"

	// According to the length of device specifications
	if len(arr) < 1 || len(arr) > 3 {
		return nil, fmt.Errorf("invalid device specification: %s", device)
	}
	src = arr[0]
	if len(arr) == 3 {
		if arr[2] != "" {
			permissions = arr[2]
		}
		dst = arr[1]
	}
	if len(arr) == 2 {
		if CheckDeviceMode(arr[1]) {
			permissions = arr[1]
		} else {
			dst = arr[1]
		}
	}

	if !CheckDeviceMode(permissions) {
		return nil, fmt.Errorf("invalid permission: %s", permissions)
	}
	if src != "" {
		if !filepath.IsAbs(src) {
			return nil, fmt.Errorf("hostpath should be an absolute path: %s", src)
		}
		src = filepath.Clean(src)
	}
	if dst != "" {
		if !filepath.IsAbs(dst) {
			return nil, fmt.Errorf("containerpath should be an absolute path: %s", dst)
		}
		dst = filepath.Clean(dst)
	}
	if src == "" && dst == "" {
		return nil, fmt.Errorf("either of host path and container path should be assigned")
	}

	ret := &types.Device{
		Path:        dst,
		PathOnHost:  src,
		Permissions: permissions,
	}

	return ret, nil
}

// CheckDeviceMode checks if the mode is ilegal.
func CheckDeviceMode(mode string) bool {
	var DeviceMode = map[rune]bool{
		'r': true,
		'w': true,
		'm': true,
	}
	if mode == "" {
		return false
	}

	for _, md := range mode {
		if !DeviceMode[md] {
			// Device Mode is ilegal
			return false
		}
		DeviceMode[md] = false
	}
	return true
}

// ParseDevice parses the device from file path and returns the device structure
func ParseDevice(device string) (*types.Device, error) {
	mapDevice, err := ParseMapping(device)
	if err != nil {
		return nil, err
	}
	dev, err := DeviceFromPath(mapDevice.PathOnHost, mapDevice.Permissions)
	if err != nil {
		return nil, err
	}
	dev.Path = mapDevice.Path
	return dev, nil
}

// GetDeviceRealPath get real path of device
func GetDeviceRealPath(path string) string {
	resolvedPathOnHost := path

	for {
		linkedPathOnHost, err := os.Readlink(resolvedPathOnHost)
		// regular file will return error
		if err != nil {
			break
		}
		base := filepath.Dir(resolvedPathOnHost)
		resolvedPathOnHost = linkedPathOnHost
		if !filepath.IsAbs(resolvedPathOnHost) {
			resolvedPathOnHost = filepath.Join(base, linkedPathOnHost)
		}
	}
	return resolvedPathOnHost
}

// GetDeviceNum get device major and minor number
func GetDeviceNum(path string) (int64, int64, error) {
	dev, err := DeviceFromPath(path, "")
	if err != nil {
		return 0, 0, err
	}

	return dev.Major, dev.Minor, nil
}

// DeviceFromPath parses the given device path to a device structure
func DeviceFromPath(path, permissions string) (*types.Device, error) {
	resolvedPathOnHost := GetDeviceRealPath(path)

	fileInfo, err := osLstat(resolvedPathOnHost)
	if err != nil {
		return nil, err
	}
	var (
		devType                string
		mode                   = fileInfo.Mode()
		fileModePermissionBits = os.FileMode.Perm(mode)
	)
	switch {
	case mode&os.ModeDevice == 0:
		return nil, ErrNotADevice
	case mode&os.ModeCharDevice != 0:
		fileModePermissionBits |= syscall.S_IFCHR
		devType = "c"
	default:
		fileModePermissionBits |= syscall.S_IFBLK
		devType = "b"
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("cannot determine the device number for device %s", path)
	}
	devNumber := int(stat.Rdev)
	return &types.Device{
		Type:        devType,
		PathOnHost:  path,
		Major:       Major(devNumber),
		Minor:       Minor(devNumber),
		Permissions: permissions,
		FileMode:    fileModePermissionBits,
		UID:         stat.Uid,
		GID:         stat.Gid,
	}, nil
}

// FindSubPartition will find all the sub-partitions for a base device.
func FindSubPartition(device *types.Device) []*types.Device {
	var subDevices []*types.Device

	cmd := exec.Command("lsblk", "-n", "-p", "-r", "-o", "NAME", device.PathOnHost)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Errorf("Failed to lsblk %s : %v", string(out), err)
		return subDevices
	}
	rawString := strings.Split(string(out), "\n")
	subDevNames := rawString[1 : len(rawString)-1]
	for _, devName := range subDevNames {
		tryDevice := devName
		if device.Path != "" {
			tryDevice = tryDevice + ":" + device.Path + string(devName[len(devName)-1])
		}

		tryDevice = tryDevice + ":" + device.Permissions

		dev, err := ParseDevice(tryDevice)
		if err != nil {
			continue
		}
		dev.Parent = device.PathOnHost
		subDevices = append(subDevices, dev)
	}
	return subDevices
}

// MknodDevice will create device in container by calling mknod system call
func MknodDevice(dest string, node *types.Device) error {
	fileMode := node.FileMode
	switch node.Type {
	case "c":
		fileMode |= syscall.S_IFCHR
	case "b":
		fileMode |= syscall.S_IFBLK
	default:
		return fmt.Errorf("%s is not a valid device type for device %s", node.Type, node.Path)
	}
	if err := syscall.Mknod(dest, uint32(fileMode), node.Mkdev()); err != nil {
		return err
	}
	return syscall.Chown(dest, int(node.UID), int(node.GID))
}

// SetDefaultPath set default path for device
func SetDefaultPath(dev *types.Device) {
	if dev.Path == "" {
		dev.Path = dev.PathOnHost
	}
	if dev.PathOnHost == "" {
		dev.PathOnHost = dev.Path
	}
}
