// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: virtual ethetic network driver
// Author: zhangwei
// Create: 2018-01-18

package veth

import (
	"fmt"
	"strings"
	"sync"

	"github.com/docker/libnetwork/netutils"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"isula.org/syscontainer-tools/libnetwork/drivers/common"
	"isula.org/syscontainer-tools/libnetwork/nsutils"
	"isula.org/syscontainer-tools/pkg/ethtool"
)

type vethDriver struct {
	*common.Driver
	veth  *netlink.Veth
	mutex sync.Mutex
}

// New will create a veth driver
func New(d *common.Driver) (*vethDriver, error) {
	driver := &vethDriver{
		Driver: d,
	}
	return driver, nil
}

func (d *vethDriver) setDefaultVethFeature(name string) error {
	etool, err := ethtool.NewEthtool(name)
	if err != nil {
		return err
	}
	defer etool.Close()
	if err := etool.SetNetDeviceTSO(true); err != nil {
		logrus.Errorf("Failed to set device %s tso on with error: %v", name, err)
	}

	if err := etool.SetNetDeviceSG(true); err != nil {
		logrus.Errorf("Failed to set device %s sg on with error: %v", name, err)
	}

	if err := etool.SetNetDeviceTX(true); err != nil {
		logrus.Errorf("Failed to set device %s tx on with error: %v", name, err)
	}

	return nil
}

func (d *vethDriver) CreateIf() error {
	logrus.Debugf("creating veth pairs")
	hostIfName, err := netutils.GenerateIfaceName("veth", 10)
	if err != nil {
		return err
	}
	guestIfName, err := netutils.GenerateIfaceName("veth", 10)
	if err != nil {
		return err
	}
	// Generate and add the interface pipe host <-> sandbox
	d.mutex.Lock()
	d.veth = &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: hostIfName, TxQLen: d.GetQlen()},
		PeerName:  guestIfName,
	}
	d.mutex.Unlock()
	if err = netlink.LinkAdd(d.veth); err != nil {
		return fmt.Errorf("failed to create veth pairs: %v", err)
	}
	if err := d.setDefaultVethFeature(guestIfName); err != nil {
		return err
	}
	if err := d.setDefaultVethFeature(hostIfName); err != nil {
		return err
	}
	logrus.Debugf("veth pair (%s, %s) created", hostIfName, guestIfName)
	return nil
}

func (d *vethDriver) DeleteIf() error {
	veth, err := netlink.LinkByName(d.GetHostNicName())
	if err != nil {
		// As add-nic supports 'update-config-only' option,
		// With this flag, syscontainer-tools will update config only, don't add device to container.
		// So if device dose not exist on host, ignore it.
		if strings.Contains(err.Error(), "Link not found") {
			return nil
		}
		return err
	}

	return netlink.LinkDel(veth)
}

func (d *vethDriver) setNicConfigure(nic netlink.Link) (rErr error) {
	// set MAC
	if d.GetMac() != nil {
		// set hardware address for interface
		if err := netlink.LinkSetHardwareAddr(nic, *(d.GetMac())); err != nil {
			return fmt.Errorf("failed to set hardware: %v", err)
		}
	}

	// set mtu
	if err := netlink.LinkSetMTU(nic, d.GetMtu()); err != nil {
		return fmt.Errorf("failed to set mtu: %v", err)
	}

	// set qlen
	if err := netlink.LinkSetTxQLen(nic, d.GetQlen()); err != nil {
		return fmt.Errorf("failed to set qlen(%d) for nic(%s)", d.GetQlen(), d.GetCtrNicName())
	}

	// set ipv4 address (TODO: ipv6 support?)
	oldAddr, _ := netlink.AddrList(nic, netlink.FAMILY_V4)
	if oldAddr != nil {
		// we only have on IP set for the interface
		if err := netlink.AddrDel(nic, &oldAddr[0]); err != nil {
			return fmt.Errorf("failed to delete old ip address: %v", err)
		}
	}
	ipAddr := &netlink.Addr{IPNet: d.GetIP(), Label: ""}
	if err := netlink.AddrAdd(nic, ipAddr); err != nil {
		return fmt.Errorf("failed to configure ip address: %v", err)
	}

	return nil
}

func (d *vethDriver) JoinAndConfigure() (rErr error) {
	if d.veth == nil || d.veth.Attrs() == nil {
		return fmt.Errorf("can't find veth interface")
	}
	// peerName does not matter, since we can delete veth pare via one end of it
	hostNicName := d.veth.Attrs().Name
	defer func() {
		if rErr != nil {
			logrus.Infof("Recover on failure: delete veth(%s)", hostNicName)
			nic, err := netlink.LinkByName(hostNicName)
			if err != nil {
				logrus.Errorf("Recover on failure: failed to get link by name(%q): %v", hostNicName, err)
				return
			}
			if err := netlink.LinkDel(nic); err != nil {
				logrus.Errorf("Recover on failure: failed to remove nic(%s): %v", hostNicName, err)
			}
		}
	}()
	err := nsutils.NsInvoke(
		d.GetNsPath(), func(nsFD int) error {
			// pre function is executed in host
			hostNic, err := netlink.LinkByName(d.veth.Attrs().Name)
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.veth.Attrs().Name, err)
			}
			ctrNic, err := netlink.LinkByName(d.veth.PeerName)
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.veth.PeerName, err)
			}

			// down the interface before configuring
			if err := netlink.LinkSetDown(hostNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}
			if err := netlink.LinkSetDown(ctrNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}

			// move the network interface to the destination
			if err := netlink.LinkSetNsFd(ctrNic, nsFD); err != nil {
				return fmt.Errorf("failed to set namespace on link %q: %v", d.veth.PeerName, err)
			}

			// attach host nic to bridge and configure mtu
			if err = netlink.LinkSetMTU(hostNic, d.GetMtu()); err != nil {
				return fmt.Errorf("failed to set mtu: %v", err)
			}

			if d.GetHostNicName() != "" {
				// set iface to user desired name
				if err := netlink.LinkSetName(hostNic, d.GetHostNicName()); err != nil {
					return fmt.Errorf("failed to rename link %s -> %s: %v", d.veth.Attrs().Name, d.GetHostNicName(), err)
				}
				hostNicName = d.GetHostNicName()
				logrus.Debugf("Rename host link %s -> %s", d.veth.Attrs().Name, d.GetHostNicName())
			}

			if err = d.AddToBridge(); err != nil {
				return fmt.Errorf("failed to add to bridge: %v", err)
			}

			if err := netlink.LinkSetUp(hostNic); err != nil {
				return fmt.Errorf("failed to set link up: %v", err)
			}
			return nil
		}, func(nsFD int) error {
			// post function is executed in container
			ctrNic, err := netlink.LinkByName(d.veth.PeerName)
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.veth.PeerName, err)
			}

			// set iface to user desired name
			if err := netlink.LinkSetName(ctrNic, d.GetCtrNicName()); err != nil {
				return fmt.Errorf("failed to rename link: %v", err)
			}
			logrus.Debugf("Rename container link %s -> %s", d.veth.PeerName, d.GetCtrNicName())

			if err := d.setNicConfigure(ctrNic); err != nil {
				return err
			}

			// Up the interface.
			if err := netlink.LinkSetUp(ctrNic); err != nil {
				return fmt.Errorf("failed to set link up: %v", err)
			}
			return nil
		})

	return err
}

func (d *vethDriver) Configure() (rErr error) {
	return nsutils.NsInvoke(
		d.GetNsPath(), func(nsFD int) error {
			// pre function is executed in host
			hostNic, err := netlink.LinkByName(d.GetHostNicName())
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.GetHostNicName(), err)
			}

			// down the interface before configuring
			if err := netlink.LinkSetDown(hostNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}

			// attach host nic to bridge and configure mtu
			if err = netlink.LinkSetMTU(hostNic, d.GetMtu()); err != nil {
				return fmt.Errorf("failed to set mtu: %v", err)
			}

			if err := netlink.LinkSetTxQLen(hostNic, d.GetQlen()); err != nil {
				return fmt.Errorf("failed to set qlen: %v", err)
			}

			if err = d.AddToBridge(); err != nil {
				return fmt.Errorf("failed to add to bridge: %v", err)
			}

			if err := netlink.LinkSetUp(hostNic); err != nil {
				return fmt.Errorf("failed to set link up: %v", err)
			}
			return nil
		}, func(nsFD int) error {
			// post function is executed in container
			ctrNic, err := netlink.LinkByName(d.GetCtrNicName())
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.GetCtrNicName(), err)
			}

			// down the interface before configuring
			if err := netlink.LinkSetDown(ctrNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}

			if err := d.setNicConfigure(ctrNic); err != nil {
				return err
			}

			// Up the interface.
			if err := netlink.LinkSetUp(ctrNic); err != nil {
				return fmt.Errorf("failed to set link up: %v", err)
			}
			return nil
		})

}

// AddTOBridge will add the veth to bridge
func (d *vethDriver) AddToBridge() error {
	if len(d.GetBridge()) == 0 {
		return fmt.Errorf("bridge can't be empty")
	}
	bd := d.GetBridgeDriver()
	if bd == nil {
		return fmt.Errorf("can't get bridge driver")
	}
	return bd.AddToBridge(d.GetHostNicName(), d.GetBridge())
}
