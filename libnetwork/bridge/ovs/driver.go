// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: ovs bridge driver implement
// Author: zhangwei
// Create: 2018-01-18

package ovs

import (
	"fmt"
	"os/exec"

	"github.com/vishvananda/netlink"
	"isula.org/isulad-tools/libnetwork/bridge/api"
)

type ovsBridgeDriver struct {
}

// Init returns the ovs bridge driver instance
func Init() api.BridgeDriver {
	return &ovsBridgeDriver{}
}

// Name returns the linux bridge driver name
func (d *ovsBridgeDriver) Name() string {
	return "ovs"
}

// AddToBridge will add an interface to bridge
func (d *ovsBridgeDriver) AddToBridge(netif, bridge string) error {
	if len(netif) == 0 || len(bridge) == 0 {
		return fmt.Errorf("bridge or network interface can't be empty")
	}

	_, err := exec.Command("ovs-vsctl", "br-exists", bridge).CombinedOutput()
	if err != nil {
		return fmt.Errorf("can't get ovs bridge %q: %v", bridge, err)
	}

	_, err = netlink.LinkByName(netif)
	if err != nil {
		return fmt.Errorf("failed to get link by name %q: %v", netif, err)
	}

	out, err := exec.Command("ovs-vsctl", "add-port", bridge, netif).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add port %q to ovs bridge %q, out: %s, err: %v", netif, bridge, out, err)
	}
	return nil
}
