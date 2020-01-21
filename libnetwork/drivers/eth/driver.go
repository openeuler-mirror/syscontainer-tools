// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: ethetic network driver
// Author: zhangwei
// Create: 2018-01-18

package eth

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/libnetwork/netutils"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"isula.org/syscontainer-tools/libnetwork/drivers/common"
	"isula.org/syscontainer-tools/libnetwork/nsutils"
	"isula.org/syscontainer-tools/pkg/ethtool"
)

type ethDriver struct {
	*common.Driver
}

// New will create a eth driver
func New(d *common.Driver) (*ethDriver, error) {
	driver := &ethDriver{
		Driver: d,
	}
	return driver, nil
}

func (d *ethDriver) setDefaultEthFeature(name string) error {
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

func (d *ethDriver) CreateIf() error {
	logrus.Debugf("creating eth device: %s:%s", d.GetHostNicName(), d.GetCtrNicName())
	if d.GetHostNicName() == "" || d.GetCtrNicName() == "" {
		return fmt.Errorf("Link name error %s:%s", d.GetHostNicName(), d.GetCtrNicName())
	}
	err := d.setDefaultEthFeature(d.GetHostNicName())
	if err != nil {
		return fmt.Errorf("failed to set default eth feature: %v", err)
	}

	hostNic, err := netlink.LinkByName(d.GetHostNicName())
	if err != nil {
		return fmt.Errorf("failed to get link by host name %q: %v", d.GetHostNicName(), err)
	}
	if err := netlink.LinkSetTxQLen(hostNic, d.GetQlen()); err != nil {
		return fmt.Errorf("failed to set qlen(%d) for nic(%s)", d.GetQlen(), d.GetHostNicName())
	}

	return nil
}

func (d *ethDriver) DeleteIf() (rErr error) {
	originalCtrNicName := d.GetCtrNicName()
	defer func() {
		if rErr != nil {
			ctrNic, err := netlink.LinkByName(d.GetCtrNicName())
			if err != nil {
				logrus.Errorf("failed to get link by name %q: %v", d.GetCtrNicName(), err)
				return
			}
			if err := netlink.LinkSetName(ctrNic, originalCtrNicName); err != nil {
				logrus.Errorf("failed to rename link: %v", err)
				return
			}
			d.Driver.SetCtrNicName(originalCtrNicName)
			logrus.Debugf("Rename link back %s -> %s", ctrNic.Attrs().Name, originalCtrNicName)
		}
	}()
	err := nsutils.NsInvoke(
		d.GetNsPath(), func(nsFD int) error {
			return nil
		}, func(nsFD int) error {
			// post function is executed in container
			ctrNic, err := netlink.LinkByName(d.GetCtrNicName())
			if err != nil {
				logrus.Warnf("%s not found", d.GetCtrNicName())
				return fmt.Errorf("failed to get link by name %q: %v", d.GetCtrNicName(), err)
			}

			logrus.Debugf("Down device %s", d.GetCtrNicName())
			// Down the interface.
			if err := netlink.LinkSetDown(ctrNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}

			initnsFD, err := os.OpenFile("/proc/1/ns/net", os.O_RDONLY, 0)
			if err != nil {
				return fmt.Errorf("failed get network namespace %q: %v", "/proc/1/ns/net", err)
			}
			defer initnsFD.Close()

			randomName, err := netutils.GenerateRandomName("tmp", 10)
			if err != nil {
				return fmt.Errorf("failed generate random name: %v", err)
			}
			if err := netlink.LinkSetName(ctrNic, randomName); err != nil {
				return fmt.Errorf("failed to rename link: %v", err)
			}
			d.Driver.SetCtrNicName(randomName)
			logrus.Debugf("Rename link %s -> %s", ctrNic.Attrs().Name, randomName)

			// move the network interface to the host namespace
			if err := netlink.LinkSetNsFd(ctrNic, int(initnsFD.Fd())); err != nil {
				return fmt.Errorf("failed to set namespace on link %q: %v", d.GetCtrNicName(), err)
			}

			return nil
		})

	if err != nil {
		if strings.Contains(err.Error(), "Link not found") {
			return nil
		}
		return err
	}

	hostNic, err := netlink.LinkByName(d.GetCtrNicName())
	if err != nil {
		return fmt.Errorf("failed to get host link by name %q: %v", d.GetCtrNicName(), err)
	}

	defer func() {
		if rErr != nil {
			// nsFD is used to recover on failure
			nsFD, err := os.OpenFile(d.GetNsPath(), os.O_RDONLY, 0)
			if err != nil {
				logrus.Errorf("Recover on failure: failed get network namespace %s: %v", d.GetNsPath(), err)
				return
			}

			// move the network interface back to the container
			if err := netlink.LinkSetNsFd(hostNic, int(nsFD.Fd())); err != nil {
				logrus.Errorf("Recover on failure: failed to move nic(%s) back to container: %v", d.GetHostNicName(), err)
				nsFD.Close()
				return
			}
			nsFD.Close()
		}
	}()
	// set iface to user desired name
	if err := netlink.LinkSetName(hostNic, d.GetHostNicName()); err != nil {
		return fmt.Errorf("failed to rename link: %v", err)
	}
	logrus.Debugf("Rename link %s -> %s", hostNic.Attrs().Name, d.GetHostNicName())

	return err
}

func (d *ethDriver) setNicConfigure(nic netlink.Link) (rErr error) {
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
	// set ipv4 address (TODO: ipv6 support?)
	ipAddr := &netlink.Addr{IPNet: d.GetIP(), Label: ""}
	if err := netlink.AddrAdd(nic, ipAddr); err != nil {
		return fmt.Errorf("failed to configure ip address: %v", err)
	}

	return nil
}

func (d *ethDriver) JoinAndConfigure() (rErr error) {
	oldName := d.GetHostNicName()

	defer func() {
		if rErr != nil {
			nic, err := netlink.LinkByName(d.GetHostNicName())
			if err != nil {
				logrus.Errorf("Recover on failure: failed to get host link by name %q: %v", d.GetHostNicName(), err)
				return
			}

			if err := netlink.LinkSetName(nic, oldName); err != nil {
				logrus.Errorf("Recover on failure: failed to rename link back to %s: %v", oldName, err)
			}
		}
	}()

	return nsutils.NsInvoke(
		d.GetNsPath(), func(nsFD int) error {
			// pre function is executed in host
			hostNic, err := netlink.LinkByName(d.GetHostNicName())
			if err != nil {
				return fmt.Errorf("failed to get link by host name %q: %v", d.GetHostNicName(), err)
			}
			randomName, err := netutils.GenerateRandomName("tmp", 10)
			if err != nil {
				return fmt.Errorf("failed generate random name: %v", err)
			}

			// down the interface before configuring
			if err := netlink.LinkSetDown(hostNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}
			logrus.Debugf("Rename link %s -> %s", hostNic.Attrs().Name, randomName)

			// set iface to user desired name
			if err := netlink.LinkSetName(hostNic, randomName); err != nil {
				return fmt.Errorf("failed to rename link: %v", err)
			}
			d.SetHostNicName(randomName)

			// move the network interface to the destination
			if err := netlink.LinkSetNsFd(hostNic, nsFD); err != nil {
				return fmt.Errorf("failed to set namespace on link %q: %v", d.GetHostNicName(), err)
			}

			return nil
		}, func(nsFD int) (rErr error) {
			// post function is executed in container
			ctrNic, err := netlink.LinkByName(d.GetHostNicName())
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.GetHostNicName(), err)
			}
			defer func() {
				if rErr != nil {
					// initnsFD is used to recover on failure
					initnsFD, err := os.OpenFile("/proc/1/ns/net", os.O_RDONLY, 0)
					if err != nil {
						logrus.Errorf("Recover on failure: failed get network namespace /proc/1/ns/net: %v", err)
						return
					}

					// move the network interface back to the host
					if err := netlink.LinkSetNsFd(ctrNic, int(initnsFD.Fd())); err != nil {
						logrus.Errorf("Recover on failure: failed to move nic(%s) back to host: %v", d.GetHostNicName(), err)
						initnsFD.Close()
						return
					}
					initnsFD.Close()
				}
			}()
			// set iface to user desired name
			if err := netlink.LinkSetName(ctrNic, d.GetCtrNicName()); err != nil {
				return fmt.Errorf("failed to rename link: %v", err)
			}
			defer func() {
				if rErr != nil {
					// still try to move this nic back to host even if recoverNicName failed
					logrus.Debugf("Recover on failure: try to rename nic name back to %s", d.GetHostNicName())
					if err := netlink.LinkSetName(ctrNic, d.GetHostNicName()); err != nil {
						d.SetHostNicName(d.GetCtrNicName())
						logrus.Errorf("Recover on failure: failed to rename nic back: %s", err)
						return
					}
					d.SetHostNicName(d.GetHostNicName())
				}
			}()

			if err = d.setNicConfigure(ctrNic); err != nil {
				return err
			}

			// Up the interface.
			if err := netlink.LinkSetUp(ctrNic); err != nil {
				return fmt.Errorf("failed to set link up: %v", err)
			}
			return nil
		})
}

func (d *ethDriver) Configure() (rErr error) {
	return nsutils.NsInvoke(
		d.GetNsPath(), func(nsFD int) error {
			return nil
		}, func(nsFD int) error {
			// post function is executed in container
			ctrNic, err := netlink.LinkByName(d.GetCtrNicName())
			if err != nil {
				return fmt.Errorf("failed to get link by name %q: %v", d.GetCtrNicName(), err)
			}

			linkUp := false
			if ctrNic.Attrs().Flags&(1<<uint(0)) == 1 {
				linkUp = true
			}

			// Down the interface before configure.
			if err := netlink.LinkSetDown(ctrNic); err != nil {
				return fmt.Errorf("failed to set link down: %v", err)
			}

			if err = d.setNicConfigure(ctrNic); err != nil {
				return err
			}

			if linkUp {
				// Up the interface.
				if err := netlink.LinkSetUp(ctrNic); err != nil {
					return fmt.Errorf("failed to set link up: %v", err)
				}
			}
			return nil
		})
}

// AddTOBridge will add the eth to bridge
func (d *ethDriver) AddToBridge() error {
	if len(d.GetBridge()) == 0 {
		return nil
	}
	return fmt.Errorf("can't add eth in container to bridge in host")
}
