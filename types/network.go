// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: network interface type
// Author: zhangwei
// Create: 2018-01-18

package types

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

// NamespacePath namespace paths
type NamespacePath struct {
	Pid  string `json:"pid,omitempty"`
	Net  string `json:"net"`
	Mnt  string `json:"mnt,omitempty"`
	User string `json:"user,omitempty"`
	Ipc  string `json:"ipc,omitempty"`
	Uts  string `json:"uts,omitempty"`
}

// InterfaceConf is the network interface config
type InterfaceConf struct {
	IP          string `json:"Ip"`
	Mac         string `json:"Mac"`
	Mtu         int    `json:"Mtu"`
	Qlen        int    `json:"Qlen"`
	Type        string `json:"Type"`
	Bridge      string `json:"Bridge"`
	HostNicName string `json:"HostNicName"`
	CtrNicName  string `json:"CtrNicName"`
}

func (nic *InterfaceConf) String() string {
	return fmt.Sprintf("Type:%s,ip:%s,name:%s->%s", nic.Type, nic.IP, nic.HostNicName, nic.CtrNicName)
}

// Route is the network route
type Route struct {
	Dest string `json:"dest"`
	Src  string `json:"src"`
	Gw   string `json:"gw"`
	Dev  string `json:"dev"`
}

// String will format the route to string format
func (r *Route) String() string {
	return fmt.Sprintf("{dest:%s,src:%s,gw:%s,dev:%s}", r.Dest, r.Src, r.Gw, r.Dev)
}

// IsConflictNic will check if the nic1 config is conflict with nic2
func IsConflictNic(nic1, nic2 *InterfaceConf) error {
	if nic1.CtrNicName == nic2.CtrNicName {
		return fmt.Errorf("interface name conflict: %s", nic1.CtrNicName)
	}
	if nic1.HostNicName == nic2.HostNicName {
		return fmt.Errorf("interface name conflict: %s", nic1.HostNicName)
	}

	if nic1.Mac != "" && (nic1.Mac == nic2.Mac) {
		return fmt.Errorf("interface mac conflict: %s", nic1.Mac)
	}
	if nic1.IP == nic2.IP {
		return fmt.Errorf("interface ip conflict: %s", nic1.IP)
	}
	return nil
}

// IsSameNic will check if the nic1 and nic2 is the same
func IsSameNic(obj, src *InterfaceConf) bool {
	if obj.IP != src.IP && obj.IP != "" {
		return false
	}
	if obj.Mac != src.Mac && obj.Mac != "" {
		return false
	}
	if obj.Mtu != src.Mtu && obj.Mtu != 0 {
		return false
	}
	if obj.Qlen != src.Qlen && obj.Qlen != 0 {
		return false
	}
	if obj.Type != src.Type && obj.Type != "" {
		return false
	}
	if obj.Bridge != src.Bridge && obj.Bridge != "" {
		return false
	}
	if obj.HostNicName != src.HostNicName && obj.HostNicName != "" {
		return false
	}
	if obj.CtrNicName != src.CtrNicName && obj.CtrNicName != "" {
		return false
	}
	return true
}

// IsConflictRoute will check if the r1 route config is conflict with r2
func IsConflictRoute(r1, r2 *Route) error {
	if IsSameRoute(r1, r2) {
		return fmt.Errorf("route %v alread exist", r1)
	}
	return nil
}

// IsSameRoute will check if the obj route config is the same with src
func IsSameRoute(obj, src *Route) bool {
	if obj.Dest != src.Dest && obj.Dest != "" {
		return false
	}
	if obj.Src != src.Src && obj.Src != "" {
		return false
	}
	if obj.Gw != src.Gw && obj.Gw != "" {
		return false
	}
	if obj.Dev != src.Dev && obj.Dev != "" {
		return false
	}
	return true
}

// ValidNetworkConfig validate network config
func ValidNetworkConfig(conf *InterfaceConf) error {
	// check IP here
	conf.IP = strings.TrimSpace(conf.IP)
	if _, err := netlink.ParseIPNet(conf.IP); err != nil {
		return err
	}

	// Check mac here
	conf.Mac = strings.TrimSpace(conf.Mac)
	if len(conf.Mac) != 0 {
		if _, err := net.ParseMAC(conf.Mac); err != nil {
			return err
		}
	}
	switch conf.Type {
	case "veth":
		if _, err := netlink.LinkByName(conf.HostNicName); err == nil {
			// found same link with hostNicName, just error out
			return fmt.Errorf("Host has nic with name %s, please choose another one", conf.HostNicName)
		}
		conf.Bridge = strings.TrimSpace(conf.Bridge)
		if conf.Bridge == "" {
			return fmt.Errorf("bridge must be specified")
		}
	case "eth":
		if conf.HostNicName == "" {
			return fmt.Errorf("host nic name input error")
		}
		if conf.Bridge != "" {
			return fmt.Errorf("for eth type, bridge cannot be set")
		}
		if _, err := netlink.LinkByName(conf.HostNicName); err != nil {
			// if HostNicName not found, just error out
			return fmt.Errorf("HostNic(%s) not found, please check", conf.HostNicName)
		}
	default:
		return fmt.Errorf("unsupported type %s", conf.Type)
	}
	return nil
}
