// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: linux bridge driver implement
// Author: zhangwei
// Create: 2018-01-18

package linux

import (
	"fmt"

	"isula.org/isulad-tools/libnetwork/bridge/api"

	"github.com/vishvananda/netlink"
)

type linuxBridgeDriver struct {
	bridge string
}

// Init returns the linux bridge driver instance
func Init() api.BridgeDriver {
	return &linuxBridgeDriver{}
}

// Name returns the linux bridge driver name
func (d *linuxBridgeDriver) Name() string {
	return "linux"
}

// AddToBridge will add an interface to bridge
func (d *linuxBridgeDriver) AddToBridge(netif, bridge string) error {
	if len(netif) == 0 || len(bridge) == 0 {
		return fmt.Errorf("bridge or network interface can't be empty")
	}
	netl, err := netlink.LinkByName(netif)
	if err != nil {
		return fmt.Errorf("failed to get link by name %q: %v", netif, err)
	}
	return netlink.LinkSetMaster(netl,
		&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridge}})
}
