// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: init bridge driver
// Author: zhangwei
// Create: 2018-01-18

package bridge

import (
	"os/exec"

	"isula.org/isulad-tools/libnetwork/bridge/api"
	"isula.org/isulad-tools/libnetwork/bridge/linux"
	"isula.org/isulad-tools/libnetwork/bridge/ovs"
)

var supportedDrivers map[string]api.BridgeDriver = make(map[string]api.BridgeDriver)

func init() {
	type initFunction func() api.BridgeDriver
	for name, initFunc := range map[string]initFunction{
		"linux": linux.Init,
		"ovs":   ovs.Init,
	} {
		supportedDrivers[name] = initFunc()
	}
}

// GetDriver will return the bridge driver by name
func GetDriver(bridgeName string) api.BridgeDriver {
	_, err := exec.Command("ovs-vsctl", "br-exists", bridgeName).CombinedOutput()
	if err == nil {
		// bridgeName is detected as an ovs bridge, return ovs driver
		return supportedDrivers["ovs"]
	}
	// error happens, use linux bridge as default:
	// 1. ovs-vsctl doesn't exist, or ovs not supported
	// 2. bridgeName isn't ovs bridge
	// whatever, fallthrough to default linux driver

	return supportedDrivers["linux"]
}
