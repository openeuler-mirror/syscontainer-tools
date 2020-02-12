// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: network config operation
// Author: zhangwei
// Create: 2018-01-18

package config

import (
	"fmt"
	"isula.org/syscontainer-tools/types"
	"path/filepath"
)

var (
	// IsuladToolsDirNetns is syscontainer-tools netns dir
	IsuladToolsDirNetns = filepath.Join(IsuladToolsDir, "netns")
)

const (
	// MaxNicNum is max nic number
	MaxNicNum = 128
)

// FindInterfaceByName will find the full config for nic by name
func (config *ContainerHookConfig) FindInterfaceByName(nic *types.InterfaceConf) *types.InterfaceConf {
	for _, eNic := range config.NetworkInterfaces {
		if nic.Type != eNic.Type && nic.Type != "" {
			continue
		}
		if nic.CtrNicName == eNic.CtrNicName && (nic.HostNicName == "" || nic.HostNicName == eNic.HostNicName) {
			return eNic
		}
		if nic.HostNicName == eNic.HostNicName && (nic.CtrNicName == "" || nic.CtrNicName == eNic.CtrNicName) {
			return eNic
		}
	}
	return nil
}

// GetNics will list all nics in config
func (config *ContainerHookConfig) GetNics(filter *types.InterfaceConf) []*types.InterfaceConf {
	interfaces := make([]*types.InterfaceConf, 0)
	for _, intf := range config.NetworkInterfaces {
		if types.IsSameNic(filter, intf) {
			interfaces = append(interfaces, intf)
		}
	}
	return interfaces
}

// IsConflictInterface will check if the new interface config is conflict with the existing ones.
func (config *ContainerHookConfig) IsConflictInterface(nic *types.InterfaceConf) error {
	for _, eNic := range config.NetworkInterfaces {
		if err := types.IsConflictNic(nic, eNic); err != nil {
			return err
		}
	}
	return nil
}

// IsSameInterface will check if the new interface config is same with the existing ones.
func (config *ContainerHookConfig) IsSameInterface(nic *types.InterfaceConf) bool {
	for _, eNic := range config.NetworkInterfaces {
		if types.IsSameNic(nic, eNic) {
			return true
		}
	}
	return false
}

// UpdateNetworkInterface will add network interface to config
func (config *ContainerHookConfig) UpdateNetworkInterface(nic *types.InterfaceConf, isAdd bool) error {
	if isAdd {
		config.dirty = true
		config.NetworkInterfaces = append(config.NetworkInterfaces, nic)
		return nil
	}

	for index, eNic := range config.NetworkInterfaces {
		if types.IsSameNic(nic, eNic) {
			config.dirty = true
			config.NetworkInterfaces = append(config.NetworkInterfaces[:index], config.NetworkInterfaces[index+1:]...)
			break
		}
	}

	return nil
}

// IsRouteExist will check if the route is added by syscontainer-tools
func (config *ContainerHookConfig) IsRouteExist(route *types.Route) bool {
	for _, eRoute := range config.NetworkRoutes {
		if types.IsSameRoute(route, eRoute) {
			return true
		}
	}

	return false
}

// IsConflictRoute will check if the new route config is conflict with the existing ones.
func (config *ContainerHookConfig) IsConflictRoute(route *types.Route) error {
	for _, eRoute := range config.NetworkRoutes {
		if err := types.IsConflictRoute(route, eRoute); err != nil {
			return err
		}
	}
	return nil
}

// GetRoutes will get all filterd routes
func (config *ContainerHookConfig) GetRoutes(filter *types.Route) []*types.Route {
	routes := make([]*types.Route, 0)
	for _, eRoute := range config.NetworkRoutes {
		if types.IsSameRoute(filter, eRoute) {
			routes = append(routes, eRoute)
		}
	}

	return routes
}

// UpdateNetworkRoutes will add route to config
func (config *ContainerHookConfig) UpdateNetworkRoutes(route *types.Route, isAdd bool) error {
	if isAdd {
		config.dirty = true
		config.NetworkRoutes = append(config.NetworkRoutes, route)
		return nil
	}

	for index, eRoute := range config.NetworkRoutes {
		if types.IsSameRoute(route, eRoute) {
			config.dirty = true
			config.NetworkRoutes = append(config.NetworkRoutes[:index], config.NetworkRoutes[index+1:]...)
			break
		}
	}

	return nil
}

// CheckNicNum check nic num reach max limit or not
func (config *ContainerHookConfig) CheckNicNum() error {
	if len(config.NetworkInterfaces) > MaxNicNum {
		return fmt.Errorf("Nic already reach max limit")
	}
	return nil
}
