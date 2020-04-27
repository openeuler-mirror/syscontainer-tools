// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: device config operation
// Author: zhangwei
// Create: 2018-01-18

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"isula.org/syscontainer-tools/types"
)

// HostMapping host path mapping to container path
type HostMapping struct {
	PathOnHost      string
	PathInContainer string
	Permission      string
}

const (
	// MaxPathNum is max path number for devices
	MaxPathNum = 128
	// ArrayLen is host bind split array len
	ArrayLen = 3
)

func parseMapping(bind string) (*HostMapping, error) {
	array := strings.SplitN(bind, ":", 3)
	if len(array) < ArrayLen {
		// this function is used for host bind.
		// should not get here, in case won't crash.
		return nil, fmt.Errorf("bind must have two : in string")
	}
	mp := &HostMapping{
		PathOnHost:      array[0],
		PathInContainer: array[1],
		Permission:      array[2],
	}
	return mp, nil
}

// Flush will flush the config to filesystem
func (config *ContainerHookConfig) Flush() error {
	if !config.dirty {
		return nil
	}
	file, err := os.Create(config.configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := file.Chmod(0600); err != nil {
		return err
	}

	if err := json.NewEncoder(file).Encode(config); err != nil {
		return err
	}
	return nil
}

func (config *ContainerHookConfig) bindIndexInArray(bind *types.Bind, array []string) int {
	for index, bindstr := range array {
		mp, err := parseMapping(bindstr)
		if err != nil {
			continue
		}
		if mp.PathInContainer == bind.ContainerPath && mp.PathOnHost == bind.HostPath {
			return index
		}
	}
	return -1
}

// DeviceIndexInArray get device index in array
func (config *ContainerHookConfig) DeviceIndexInArray(device *types.Device) int {

	for index, dev := range config.Devices {
		if (device.PathOnHost == "" && dev.PathInContainer == device.Path) ||
			(device.Path == "" && dev.PathOnHost == device.PathOnHost) ||
			(dev.PathInContainer == device.Path && dev.PathOnHost == device.PathOnHost) {
			return index
		}
	}
	return -1
}

func (config *ContainerHookConfig) getConflictIndex(device *types.Device) int {

	for index, dev := range config.Devices {
		if dev.PathInContainer == device.Path || dev.PathOnHost == device.PathOnHost {
			return index
		}
	}
	return -1
}

// FindDeviceByMapping returns if a device in DeviceToAdd Config.
func (config *ContainerHookConfig) FindDeviceByMapping(device *types.Device) *types.Device {
	for _, eDevice := range config.Devices {
		if eDevice.PathOnHost == device.PathOnHost && eDevice.PathInContainer == device.Path {
			return &types.Device{
				Path:        eDevice.PathInContainer,
				PathOnHost:  eDevice.PathOnHost,
				Permissions: eDevice.CgroupPermissions,
				Major:       eDevice.Major,
				Minor:       eDevice.Minor,
				Type:        eDevice.Type,
				Parent:      eDevice.Parent,
			}
		}
	}
	return nil
}

// FindSubPartition returns a set of sub devices of a device by config.
func (config *ContainerHookConfig) FindSubPartition(device *types.Device) []*types.Device {
	var ret []*types.Device
	for _, eDevice := range config.Devices {
		if eDevice.Parent == device.PathOnHost {
			ret = append(ret, &types.Device{
				Path:        eDevice.PathInContainer,
				PathOnHost:  eDevice.PathOnHost,
				Permissions: eDevice.CgroupPermissions,
				Major:       eDevice.Major,
				Minor:       eDevice.Minor,
				Type:        eDevice.Type,
				Parent:      eDevice.Parent,
			})
		}
	}
	return ret
}

// UpdateDevice will update hook config of devices
func (config *ContainerHookConfig) UpdateDevice(device *types.Device, isAddDevice bool) error {
	dev := &DeviceMapping{
		Type:              device.Type,
		Major:             device.Major,
		Minor:             device.Minor,
		PathOnHost:        device.PathOnHost,
		PathInContainer:   device.Path,
		CgroupPermissions: device.Permissions,
		Parent:            device.Parent,
	}

	// add device action:
	if isAddDevice {
		// if not exist in Add array, add it.
		index := config.getConflictIndex(device)
		if index != -1 {
			conflictDev := config.Devices[index]
			return fmt.Errorf("device %s:%s has been already added into container", conflictDev.PathOnHost, conflictDev.PathInContainer)
		}
		config.dirty = true
		config.Devices = append(config.Devices, dev)
	} else {
		if index := config.DeviceIndexInArray(device); index != -1 {
			config.dirty = true
			config.Devices = append(config.Devices[:index], config.Devices[index+1:]...)
		} else {
			return fmt.Errorf("device %s:%s has not been added into container", device.PathOnHost, device.Path)
		}
	}
	return nil

}

// IsBindInConfig returns if a device in DeviceToAdd Config.
func (config *ContainerHookConfig) IsBindInConfig(bind *types.Bind) bool {
	if index := config.bindIndexInArray(bind, config.Binds); index != -1 {
		return true
	}
	return false
}

// GetBindInConfig returns device in DeviceToAdd Config.
func (config *ContainerHookConfig) GetBindInConfig(bind *types.Bind) (*HostMapping, error) {
	if index := config.bindIndexInArray(bind, config.Binds); index != -1 {
		return parseMapping(config.Binds[index])
	}
	return nil, fmt.Errorf("fail to find bind: %v", bind)
}

func (config *ContainerHookConfig) addBind(bind *types.Bind) (bool, error) {
	// if not in Add array, add it.
	exist, err := config.bi.add(bind.ToString())
	if err != nil {
		return exist, err
	}
	if index := config.bindIndexInArray(bind, config.Binds); index == -1 {
		config.dirty = true
		config.Binds = append(config.Binds, bind.ToString())
	}
	return exist, nil

}
func (config *ContainerHookConfig) removeBind(bind *types.Bind) (bool, error) {
	// if in add array, remove it, will not restore to Rm array, as it is added by syscontainer-tools.
	if index := config.bindIndexInArray(bind, config.Binds); index != -1 {
		config.dirty = true
		config.Binds = append(config.Binds[:index], config.Binds[index+1:]...)
	}

	return config.bi.remove(bind.ToString())

}

// UpdateBind will update binds of hook config
func (config *ContainerHookConfig) UpdateBind(bind *types.Bind, isAddBind bool) (bool, error) {
	if isAddBind {
		return config.addBind(bind)
	}
	return config.removeBind(bind)
}

// GetBinds get binds of hook config
func (config *ContainerHookConfig) GetBinds() []string {
	return config.Binds[:]
}

// GetAllDevices get all devices of hook config
func (config *ContainerHookConfig) GetAllDevices() []*DeviceMapping {
	return config.Devices[:]
}

// UpdateDeviceQos will update the qos for device
func (config *ContainerHookConfig) UpdateDeviceQos(qos *types.Qos, qType QosType) error {
	update := func(qosArr []*types.Qos, qos *types.Qos) []*types.Qos {
		for _, q := range qosArr {
			if q.Major == qos.Major && q.Minor == qos.Minor {
				if q.Value != qos.Value {
					config.dirty = true
					q.Value = qos.Value
				}
				return qosArr
			}
		}
		config.dirty = true
		qosArr = append(qosArr, qos)
		return qosArr
	}
	switch qType {
	case QosReadIOPS:
		config.ReadIOPS = update(config.ReadIOPS, qos)
	case QosWriteIOPS:
		config.WriteIOPS = update(config.WriteIOPS, qos)
	case QosReadBps:
		config.ReadBps = update(config.ReadBps, qos)
	case QosWriteBps:
		config.WriteBps = update(config.WriteBps, qos)
	case QosBlkioWeight:
		config.BlkioWeight = update(config.BlkioWeight, qos)
	}
	return nil
}

// RemoveDeviceQos remove qos for device
func (config *ContainerHookConfig) RemoveDeviceQos(device *types.Device, qType QosType) (bool, error) {
	remove := func(qosArr []*types.Qos, device *types.Device) ([]*types.Qos, bool) {
		for index, q := range qosArr {
			if q.Major == device.Major && q.Minor == device.Minor {
				config.dirty = true
				qosArr = append(qosArr[:index], qosArr[index+1:]...)
				return qosArr, true
			}
		}
		return qosArr, false
	}
	var ret bool
	switch qType {
	case QosReadIOPS:
		config.ReadIOPS, ret = remove(config.ReadIOPS, device)
	case QosWriteIOPS:
		config.WriteIOPS, ret = remove(config.WriteIOPS, device)
	case QosReadBps:
		config.ReadBps, ret = remove(config.ReadBps, device)
	case QosWriteBps:
		config.WriteBps, ret = remove(config.WriteBps, device)
	case QosBlkioWeight:
		config.BlkioWeight, ret = remove(config.BlkioWeight, device)
	}
	return ret, nil
}

// CheckPathNum check path num reach max limit or not
func (config *ContainerHookConfig) CheckPathNum() error {
	if len(config.Binds) > MaxPathNum {
		return fmt.Errorf("Path already reach max limit")
	}
	return nil
}

// SetConfigDirty set config dir dirty
func (config *ContainerHookConfig) SetConfigDirty() {
	config.dirty = true
}

// UpdateQosDevNum update qos major and minor for device
func (config *ContainerHookConfig) UpdateQosDevNum(qos *types.Qos, major int64, minor int64) {
	if qos.Major != major || qos.Minor != minor {
		qos.Major = major
		qos.Minor = minor
		config.dirty = true
	}
}

// UpdateDeviceNode update device node
func (config *ContainerHookConfig) UpdateDeviceNode(device string, major, minor int64) {
	for index, dev := range config.Devices {
		if dev.PathOnHost == device {
			config.Devices[index].Major = major
			config.Devices[index].Minor = minor
			config.dirty = true
		}
	}

	for index, qos := range config.ReadIOPS {
		if qos.Path == device {
			config.ReadIOPS[index].Major = major
			config.ReadIOPS[index].Minor = minor
			config.dirty = true
		}
	}

	for index, qos := range config.WriteIOPS {
		if qos.Path == device {
			config.WriteIOPS[index].Major = major
			config.WriteIOPS[index].Minor = minor
			config.dirty = true
		}
	}

	for index, qos := range config.ReadBps {
		if qos.Path == device {
			config.ReadBps[index].Major = major
			config.ReadBps[index].Minor = minor
			config.dirty = true
		}
	}

	for index, qos := range config.WriteBps {
		if qos.Path == device {
			config.WriteBps[index].Major = major
			config.WriteBps[index].Minor = minor
			config.dirty = true
		}
	}

	for index, qos := range config.BlkioWeight {
		if qos.Path == device {
			config.BlkioWeight[index].Major = major
			config.BlkioWeight[index].Minor = minor
			config.dirty = true
		}
	}
}
