// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: network interface operation
// Author: zhangwei
// Create: 2018-01-18

package libnetwork

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	hconfig "isula.org/isulad-tools/config"
	"isula.org/isulad-tools/container"
	"isula.org/isulad-tools/libnetwork/drivers"
	"isula.org/isulad-tools/types"
)

// AddNic will add a network interface to container, it will update the config for container
func AddNic(ctr *container.Container, config *types.InterfaceConf, updateConfigOnly bool) error {
	if err := ctr.Lock(); err != nil {
		return err
	}
	defer ctr.Unlock()
	// create config file handler.
	hConfig, err := hconfig.NewContainerConfig(ctr)
	if err != nil {
		return err
	}
	defer hConfig.Flush()

	if err := hConfig.CheckNicNum(); err != nil {
		return err
	}

	if err := hConfig.IsConflictInterface(config); err != nil {
		return err
	}
	if err := hConfig.UpdateNetworkInterface(config, true); err != nil {
		return err
	}
	// don't insert net interface when:
	// 1. update-config-only flag is set
	// 2. container isn't running(pid==0)
	if !updateConfigOnly && ctr.Pid() > 0 && ctr.CheckPidExist() {
		if err := AddNicToContainer(ctr.NetNsPath(), config); err != nil {
			// roll back
			hConfig.UpdateNetworkInterface(config, false)
			return err
		}
	}
	fmt.Fprintf(os.Stdout, "Add network interface (%s) to container (%s,%s) done\n", config.HostNicName, ctr.Name(), config.CtrNicName)
	logrus.Infof("Add network interface (%s) to container (%s,%s) done", config.HostNicName, ctr.Name(), config.CtrNicName)
	return nil
}

// AddNicToContainer will add a network interface to container only.
// It will be called by network-hook
func AddNicToContainer(nsPath string, config *types.InterfaceConf) (rErr error) {
	driver, err := drivers.New(config.Type,
		drivers.NicOptionCtrNicName(config.CtrNicName),
		drivers.NicOptionHostNicName(config.HostNicName),
		drivers.NicOptionNsPath(nsPath),
		drivers.NicOptionIP(config.IP),
		drivers.NicOptionMac(config.Mac),
		drivers.NicOptionMtu(config.Mtu),
		drivers.NicOptionQlen(config.Qlen),
		drivers.NicOptionBridge(config.Bridge))
	if err != nil {
		return err
	}

	if err := driver.CreateIf(); err != nil {
		return fmt.Errorf("failed to create interface: %v", err)
	}
	// do not need to DeleteIf here, if CreateIf failed, there are no ifs.
	// JoinAndConfigure is doing cleanup within itself
	return driver.JoinAndConfigure()
}

// UpdateNicInContainer will update an existing  network interface in container.
func UpdateNicInContainer(nsPath string, config *types.InterfaceConf) (rErr error) {
	driver, err := drivers.New(config.Type,
		drivers.NicOptionCtrNicName(config.CtrNicName),
		drivers.NicOptionHostNicName(config.HostNicName),
		drivers.NicOptionNsPath(nsPath),
		drivers.NicOptionIP(config.IP),
		drivers.NicOptionMac(config.Mac),
		drivers.NicOptionMtu(config.Mtu),
		drivers.NicOptionQlen(config.Qlen),
		drivers.NicOptionBridge(config.Bridge))
	if err != nil {
		return err
	}

	// Configure is doing cleanup within itself
	return driver.Configure()
}

// DelNic will remove a network interface from container and update the config
func DelNic(ctr *container.Container, config *types.InterfaceConf) error {
	if err := ctr.Lock(); err != nil {
		return err
	}
	defer ctr.Unlock()
	// create config file handler.
	hConfig, err := hconfig.NewContainerConfig(ctr)
	if err != nil {
		return err
	}
	defer hConfig.Flush()

	var newConfig *types.InterfaceConf
	if newConfig = hConfig.FindInterfaceByName(config); newConfig == nil {
		return fmt.Errorf("Network interface %s,%s with type %s not exist in container %s", config.HostNicName, config.CtrNicName, config.Type, ctr.Name())
	}
	if err := hConfig.UpdateNetworkInterface(newConfig, false); err != nil {
		return err
	}
	// only work for running container
	if ctr.Pid() > 0 && ctr.CheckPidExist() {
		if err := DelNicFromContainer(ctr.NetNsPath(), newConfig); err != nil {
			if !strings.Contains(err.Error(), "failed to get host link by name") {
				// roll back
				hConfig.UpdateNetworkInterface(newConfig, true)
				return err
			}
			logrus.Errorf("Remove network interface error: %s", err)
		}
	}

	fmt.Fprintf(os.Stdout, "Remove network interface (%s) from container (%s,%s) done\n", newConfig.HostNicName, ctr.Name(), newConfig.CtrNicName)
	logrus.Infof("Remove network interface (%s) from container (%s,%s) done", newConfig.HostNicName, ctr.Name(), newConfig.CtrNicName)

	return nil
}

// DelNicFromContainer will remove a network interface from container only
func DelNicFromContainer(nsPath string, config *types.InterfaceConf) error {
	driver, err := drivers.New(config.Type,
		drivers.NicOptionCtrNicName(config.CtrNicName),
		drivers.NicOptionHostNicName(config.HostNicName),
		drivers.NicOptionNsPath(nsPath),
		drivers.NicOptionIP(config.IP),
		drivers.NicOptionMac(config.Mac),
		drivers.NicOptionMtu(config.Mtu),
		drivers.NicOptionBridge(config.Bridge))

	if err != nil {
		return err
	}

	return driver.DeleteIf()
}

// UpdateNic will reconfigure network interface for a container
func UpdateNic(ctr *container.Container, config *types.InterfaceConf, updateConfigOnly bool) error {
	if err := ctr.Lock(); err != nil {
		return err
	}
	defer ctr.Unlock()
	hConfig, err := hconfig.NewContainerConfig(ctr)
	if err != nil {
		return err
	}

	var tmpConfig = new(types.InterfaceConf)
	tmpConfig.CtrNicName = config.CtrNicName

	var newConfig *types.InterfaceConf
	if newConfig = hConfig.FindInterfaceByName(tmpConfig); newConfig == nil {
		return fmt.Errorf("Network interface %s,%s with type %s not exist in container %s", config.HostNicName, config.CtrNicName, config.Type, ctr.Name())
	}

	if config.IP == "" {
		tmpConfig.IP = newConfig.IP
	} else {
		tmpConfig.IP = config.IP
		msg := fmt.Sprintf("Update IP address for network interface (%s,%v) done", config.CtrNicName, config.IP)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)
	}
	if config.Mac == "" {
		tmpConfig.Mac = newConfig.Mac
	} else {
		tmpConfig.Mac = config.Mac
		msg := fmt.Sprintf("Update MAC address for network interface (%s,%v) done", config.CtrNicName, config.Mac)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	if config.Bridge == "" {
		tmpConfig.Bridge = newConfig.Bridge
	} else {
		tmpConfig.Bridge = config.Bridge
		msg := fmt.Sprintf("Update Bridge for network interface (%s,%v) done", config.CtrNicName, config.Bridge)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	if config.Mtu == 0 {
		tmpConfig.Mtu = newConfig.Mtu
	} else {
		tmpConfig.Mtu = config.Mtu
		msg := fmt.Sprintf("Update Mtu for network interface (%s,%v) done", config.CtrNicName, config.Mtu)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)

	}
	// we use qlen < 0 to check if the user has set parameter qlen or not
	if config.Qlen < 0 {
		tmpConfig.Qlen = newConfig.Qlen
	} else {
		tmpConfig.Qlen = config.Qlen
		msg := fmt.Sprintf("Update Qlen for network interface (%s,%v)", config.CtrNicName, config.Qlen)
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)
	}
	tmpConfig.Type = newConfig.Type
	tmpConfig.HostNicName = newConfig.HostNicName

	if hConfig.IsSameInterface(tmpConfig) {
		logrus.Infof("Network interface in container %s: Identical setting, nothing to change", config.CtrNicName, ctr.Name())
		return nil
	}
	if err := hConfig.UpdateNetworkInterface(newConfig, false); err != nil {
		return err
	}
	if err := hConfig.IsConflictInterface(tmpConfig); err != nil {
		if err := hConfig.UpdateNetworkInterface(newConfig, true); err != nil {
			return err
		}
		return err
	}

	if !updateConfigOnly && ctr.Pid() > 0 && ctr.CheckPidExist() {
		if err := UpdateNicInContainer(ctr.NetNsPath(), tmpConfig); err != nil {
			if err := hConfig.UpdateNetworkInterface(newConfig, true); err != nil {
				return err
			}
			return err
		}
	}

	// update the config file.
	if err := hConfig.UpdateNetworkInterface(tmpConfig, true); err != nil {
		return err
	}
	hConfig.Flush()
	logrus.Infof("Network interface %s in container %s update successfully", config.CtrNicName, ctr.Name())
	return nil
}

// ListNic will list all network interfaces in a container
func ListNic(ctr *container.Container, filter *types.InterfaceConf) ([]*types.InterfaceConf, error) {
	if err := ctr.Lock(); err != nil {
		return nil, err
	}
	defer ctr.Unlock()
	hConfig, err := hconfig.NewContainerConfig(ctr)
	if err != nil {
		return nil, err
	}

	return hConfig.GetNics(filter), nil
}
