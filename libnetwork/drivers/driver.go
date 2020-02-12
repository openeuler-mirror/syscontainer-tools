// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: network interface driver
// Author: zhangwei
// Create: 2018-01-18

package drivers

import (
	"fmt"
	"net"
	"strings"

	// "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"isula.org/syscontainer-tools/libnetwork/drivers/common"
	"isula.org/syscontainer-tools/libnetwork/drivers/eth"
	"isula.org/syscontainer-tools/libnetwork/drivers/veth"
)

var (
	// ErrTypeNotSupported is the interface type not supported error
	ErrTypeNotSupported = fmt.Errorf("network interface type not supported")
)

const (
	// OptionMtu mtu must not be less than 68
	OptionMtu = 68
)

// Driver defines the network driver function interface
type Driver interface {
	// CreateNic create interface according to type
	CreateIf() error
	// DeleteIf delete interface from container
	DeleteIf() error
	// JoinAndConfigure join network interface into namespace and configure it
	JoinAndConfigure() error
	// Configure  update network interface in a container
	Configure() error
	// AddToBridge adds interface to bridge
	AddToBridge() error
}

// New will crate a network driver by type and options
func New(driverType string, options ...DriverOptions) (Driver, error) {
	d := &common.Driver{}
	if err := processOptions(d, options...); err != nil {
		return nil, err
	}

	switch driverType {
	case "", "veth":
		return veth.New(d)
	case "eth":
		return eth.New(d)
	case "sriov", "dpdk":
		fallthrough
	default:
		return nil, ErrTypeNotSupported
	}
}

// DriverOptions define a callback function to handle driver option
type DriverOptions func(d *common.Driver) error

func processOptions(d *common.Driver, options ...DriverOptions) error {
	for _, op := range options {
		if err := op(d); err != nil {
			return err
		}
	}
	return nil
}

// NicOptionNsPath handles network namespace path option
func NicOptionNsPath(nsPath string) DriverOptions {
	return func(d *common.Driver) error {
		nsPath = strings.TrimSpace(nsPath)
		if len(nsPath) != 0 {
			d.SetNsPath(nsPath)
		}
		return nil
	}
}

// NicOptionCtrNicName handles interface name in container option
func NicOptionCtrNicName(name string) DriverOptions {
	return func(d *common.Driver) error {
		d.SetCtrNicName(strings.TrimSpace(name))
		return nil
	}
}

// NicOptionHostNicName handles the network interface name on host opstion
func NicOptionHostNicName(name string) DriverOptions {
	return func(d *common.Driver) error {
		d.SetHostNicName(strings.TrimSpace(name))
		return nil
	}
}

// NicOptionIP handles network interface ip option
func NicOptionIP(ip string) DriverOptions {
	return func(d *common.Driver) error {
		ip = strings.TrimSpace(ip)
		ipnet, err := netlink.ParseIPNet(ip)
		if err != nil {
			return err
		}

		d.SetIP(ipnet)
		return nil
	}
}

// NicOptionMac handles network interface mac option
func NicOptionMac(mac string) DriverOptions {
	return func(d *common.Driver) error {
		if len(strings.TrimSpace(mac)) == 0 {
			return nil
		}
		hw, err := net.ParseMAC(strings.TrimSpace(mac))
		if err != nil {
			return err
		}
		d.SetMac(&hw)
		return nil
	}
}

// NicOptionMtu handles interface mtu option
func NicOptionMtu(mtu int) DriverOptions {
	return func(d *common.Driver) error {
		if mtu < OptionMtu {
			return fmt.Errorf("Mtu must not be less than 68")
		}
		d.SetMtu(mtu)
		return nil
	}
}

// NicOptionQlen handles interface Qlen option
func NicOptionQlen(qlen int) DriverOptions {
	return func(d *common.Driver) error {
		if qlen < 0 {
			return fmt.Errorf("Qlen must not be less than 0")
		}
		d.SetQlen(qlen)
		return nil
	}
}

// NicOptionBridge handles brigde name option
func NicOptionBridge(bridge string) DriverOptions {
	return func(d *common.Driver) error {
		d.SetBridge(strings.TrimSpace(bridge))
		return nil
	}
}
