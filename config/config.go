// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: device hook config
// Author: zhangwei
// Create: 2018-01-18

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"isula.org/isulad-tools/container"
	"isula.org/isulad-tools/types"
)

const (
	defaultConfigFile = "device_hook.json"
	// IsuladToolsDir is isulad-tools run dir
	IsuladToolsDir = "/run/isulad-tools"
)

// QosType defines the qos type by int
type QosType int

const (
	// QosReadIOPS defines the read device iops type
	QosReadIOPS QosType = iota
	// QosWriteIOPS defines the write device iops type
	QosWriteIOPS
	// QosReadBps defines the read device bps type
	QosReadBps
	// QosWriteBps defines the write device bps type
	QosWriteBps
	// QosBlkioWeight defines the device blkio weight type
	QosBlkioWeight
)

// ContainerConfig is the interface of container config handler
type ContainerConfig interface {
	FindDeviceByMapping(dev *types.Device) *types.Device
	FindSubPartition(dev *types.Device) []*types.Device
	UpdateDevice(device *types.Device, isAddDevice bool) error
	UpdateDeviceNode(device string, major, minor int64)

	IsBindInConfig(bind *types.Bind) bool
	UpdateBind(bind *types.Bind, isAddBind bool) (bool, error)
	GetBinds() []string
	GetBindInConfig(bind *types.Bind) (*HostMapping, error)
	GetAllDevices() []*DeviceMapping
	DeviceIndexInArray(device *types.Device) int
	UpdateDeviceQos(qos *types.Qos, qType QosType) error
	RemoveDeviceQos(device *types.Device, qType QosType) (bool, error)

	FindInterfaceByName(config *types.InterfaceConf) *types.InterfaceConf
	IsConflictInterface(nic *types.InterfaceConf) error
	IsSameInterface(nic *types.InterfaceConf) bool
	UpdateNetworkInterface(nic *types.InterfaceConf, isAdd bool) error
	GetNics(filter *types.InterfaceConf) []*types.InterfaceConf

	IsRouteExist(route *types.Route) bool
	IsConflictRoute(route *types.Route) error
	UpdateNetworkRoutes(route *types.Route, isAdd bool) error
	GetRoutes(filter *types.Route) []*types.Route

	Flush() error
	CheckPathNum() error
	CheckNicNum() error
}

// NewContainerConfig will create the container config handler by name
func NewContainerConfig(c *container.Container) (ContainerConfig, error) {
	configfile := filepath.Join(c.ContainerPath(), defaultConfigFile)
	hConfig, err := LoadContainerHookConfig(configfile)
	if err != nil {
		return nil, err
	}
	hConfig.configPath = configfile
	return hConfig, nil
}

// DeviceMapping represents the device mapping between the host and the container.
type DeviceMapping struct {
	Type              string
	Minor             int64
	Major             int64
	PathOnHost        string
	PathInContainer   string
	CgroupPermissions string
	Parent            string
}

type info struct {
	count int
	perm  string
}
type bindsInfo struct {
	pathInHost      map[string]*info
	pathInContainer map[string]int
	l               *sync.Mutex
}

func checkEuqal(old, new string) bool {
	oldp := strings.Split(strings.Replace(old, " ", "", -1), ",")
	newp := strings.Split(strings.Replace(new, " ", "", -1), ",")
	return reflect.DeepEqual(oldp, newp)

}

// we do not allow mount more than on host paths to a single path in container
// if we mount a single host path to multi paths in contiainer, return true,nil
func (bi *bindsInfo) add(bindstr string) (bool, error) {

	bi.l.Lock()
	defer bi.l.Unlock()

	hostPathExist := false

	mp, err := parseMapping(bindstr)
	if err != nil {
		return hostPathExist, fmt.Errorf("Wrong bind format: %s,err %s", bindstr, err)
	}

	if _, exist := bi.pathInContainer[mp.PathInContainer]; exist == true {
		return hostPathExist, fmt.Errorf("Mount more than one host paths to a single path in container")
	}

	bi.pathInContainer[mp.PathInContainer] = 1

	if _, exist := bi.pathInHost[mp.PathOnHost]; exist == true {
		if checkEuqal(mp.Permission, bi.pathInHost[mp.PathOnHost].perm) == false {
			return hostPathExist, fmt.Errorf("Mount one host path with different permissions, old: %s, new: %s", bi.pathInHost[mp.PathOnHost].perm, mp.Permission)
		}
		bi.pathInHost[mp.PathOnHost].count++
		hostPathExist = true
		return hostPathExist, nil
	}
	bi.pathInHost[mp.PathOnHost] = &info{count: 1, perm: mp.Permission}
	return hostPathExist, nil
}

func (bi *bindsInfo) remove(bindstr string) (bool, error) {
	bi.l.Lock()
	defer bi.l.Unlock()
	mp, err := parseMapping(bindstr)
	if err != nil {
		return true, fmt.Errorf("Wrong bind format: %s,err %s", bindstr, err)
	}

	// always delete the item of container path
	delete(bi.pathInContainer, mp.PathInContainer)

	if _, exist := bi.pathInHost[mp.PathOnHost]; exist == true {
		bi.pathInHost[mp.PathOnHost].count--
		if bi.pathInHost[mp.PathOnHost].count <= 0 {
			delete(bi.pathInHost, mp.PathOnHost)
			return true, nil
		}
		return false, nil
	}
	return true, fmt.Errorf("%s not in memory datebase", mp.PathOnHost)

}

// ContainerHookConfig is the data config structure for device hook storage file.
type ContainerHookConfig struct {
	Binds             []string               `json:"bindToAdd,omitempty"`
	Devices           []*DeviceMapping       `json:"deviceToAdd,omitempty"`
	ReadIOPS          []*types.Qos           `json:"readIops,omitempty"`
	WriteIOPS         []*types.Qos           `json:"writeIops,omitempty"`
	ReadBps           []*types.Qos           `json:"readBps,omitempty"`
	WriteBps          []*types.Qos           `json:"writeBps,omitempty"`
	BlkioWeight       []*types.Qos           `json:"blkioWeight,omitempty"`
	NetworkInterfaces []*types.InterfaceConf `json:"networkInterfaces,omitempty"`
	NetworkRoutes     []*types.Route         `json:"networkRoute,omitempty"`
	configPath        string
	dirty             bool
	bi                *bindsInfo
}

// LoadContainerHookConfig will parse and unmarshal ContainerHookConfig
func LoadContainerHookConfig(path string) (*ContainerHookConfig, error) {
	// if config file do not exist, just return empty DevieHookConfig.
	bi := bindsInfo{
		pathInHost:      make(map[string]*info),
		pathInContainer: make(map[string]int),
		l:               &sync.Mutex{},
	}
	if _, err := os.Stat(path); err != nil {
		return &ContainerHookConfig{bi: &bi}, nil
	}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := &ContainerHookConfig{}

	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}
	config.bi = &bi

	for _, bindstr := range config.Binds {
		if _, err := config.bi.add(bindstr); err != nil {
			return nil, err
		}
	}

	config.configPath = path

	return config, nil
}
