// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: common network driver
// Author: zhangwei
// Create: 2018-01-18

package common

import (
	"net"

	"isula.org/syscontainer-tools/libnetwork/bridge"
	"isula.org/syscontainer-tools/libnetwork/bridge/api"
)

// Driver implement the network driver common options
type Driver struct {
	nsPath       string
	ctrName      string
	hostName     string
	mac          *net.HardwareAddr
	ip           *net.IPNet
	bridge       string
	bridgeDriver api.BridgeDriver
	mtu          int
	qlen         int
}

// SetCtrNicName will set the network interface name in container
func (d *Driver) SetCtrNicName(name string) {
	d.ctrName = name
}

// GetCtrNicName will return the network interface name in container
func (d *Driver) GetCtrNicName() string {
	return d.ctrName
}

// SetHostNicName will set the network interface name on host
func (d *Driver) SetHostNicName(name string) {
	d.hostName = name
}

// GetHostNicName will return the network interface name on host
func (d *Driver) GetHostNicName() string {
	return d.hostName
}

// SetNsPath will set the network namespace path
func (d *Driver) SetNsPath(path string) {
	d.nsPath = path
}

// GetNsPath will return the network namespace path
func (d *Driver) GetNsPath() string {
	return d.nsPath
}

// SetIP will set the network interface ip
func (d *Driver) SetIP(addr *net.IPNet) {
	d.ip = addr
}

// GetIP will set the network interface ip
func (d *Driver) GetIP() *net.IPNet {
	return d.ip
}

// SetMac will set the network interface mac
func (d *Driver) SetMac(mac *net.HardwareAddr) {
	d.mac = mac
}

// GetMac will return the network interface mac
func (d *Driver) GetMac() *net.HardwareAddr {
	return d.mac
}

// SetMtu will set the network interface mtu
func (d *Driver) SetMtu(mtu int) {
	d.mtu = mtu
}

// GetMtu will return the network interface mtu
func (d *Driver) GetMtu() int {
	return d.mtu
}

// SetQlen will set the network interface qlen
func (d *Driver) SetQlen(qlen int) {
	d.qlen = qlen
}

// GetQlen will return the network interface qlen
func (d *Driver) GetQlen() int {
	return d.qlen
}

// SetBridge will set the bridge name which the nic connected to
func (d *Driver) SetBridge(bridgeName string) {
	d.bridge = bridgeName
	d.bridgeDriver = bridge.GetDriver(bridgeName)
}

// GetBridge will return the bridge name which the nic connected to
func (d *Driver) GetBridge() string {
	return d.bridge
}

// GetBridgeDriver will return bridge driver interface
func (d *Driver) GetBridgeDriver() api.BridgeDriver {
	return d.bridgeDriver
}
