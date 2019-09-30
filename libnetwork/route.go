// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: network routes operation
// Author: zhangwei
// Create: 2018-01-18

package libnetwork

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	hconfig "isula.org/isulad-tools/config"
	"isula.org/isulad-tools/container"
	"isula.org/isulad-tools/libnetwork/nsutils"
	"isula.org/isulad-tools/types"

	"github.com/vishvananda/netlink"
)

// AddRoutes will add network routes to contianer and update container config.
func AddRoutes(ctr *container.Container, routes []*types.Route, updateConfigOnly bool) error {
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

	for _, route := range routes {
		if err := hConfig.IsConflictRoute(route); err != nil {
			return err
		}
		if err := hConfig.UpdateNetworkRoutes(route, true); err != nil {
			return err
		}
		// Don't insert real route rules when:
		// 1. update-config-only flag is set
		// 2. or container isn't running(pid=0)
		if !updateConfigOnly && ctr.Pid() > 0 && ctr.CheckPidExist() {
			if err := AddRouteToContainer(ctr.NetNsPath(), route); err != nil {
				// roll back
				hConfig.UpdateNetworkRoutes(route, false)
				return err
			}
		}
		msg := fmt.Sprintf("Add route to container %s, route: %s done", ctr.Name(), route.String())
		fmt.Fprintln(os.Stdout, msg)
		logrus.Info(msg)
	}
	return nil
}

// AddRouteToContainer will add one route to container.
// It will be called by network-hook too.
func AddRouteToContainer(nsPath string, route *types.Route) error {
	var err error
	src := strings.TrimSpace(route.Src)
	dest := strings.TrimSpace(route.Dest)
	gw := strings.TrimSpace(route.Gw)
	dev := strings.TrimSpace(route.Dev)
	if len(src) == 0 && len(gw) == 0 && len(dev) == 0 {
		return fmt.Errorf("src or gw or dev name is required")
	}

	rule := &netlink.Route{}
	if dest == "default" {
		dest = ""
	}
	if len(dest) != 0 {
		rule.Dst, err = netlink.ParseIPNet(dest)
		if err != nil {
			return fmt.Errorf("failed to parse dest %q of route rule", dest)
		}
	}

	if len(src) != 0 {
		rule.Src = net.ParseIP(src)
		if err != nil {
			return fmt.Errorf("failed to parse src ip")
		}
	}

	if len(gw) != 0 {
		rule.Gw = net.ParseIP(gw)
		if err != nil {
			return fmt.Errorf("failed to parse gw ip")
		}
	}

	return nsutils.NsInvoke(nsPath,
		func(nsFD int) error { return nil },
		func(nsFD int) error {
			// executed in container
			if len(dev) != 0 {
				ctrNic, err := netlink.LinkByName(dev)
				if err != nil || ctrNic == nil {
					return fmt.Errorf("failed to get link by name %s: %v", dev, err)
				}
				rule.LinkIndex = ctrNic.Attrs().Index
			}

			if err := netlink.RouteAdd(rule); err != nil {
				return fmt.Errorf("failed to add route: %v", err)
			}
			return nil
		})
}

// DelRoutes will remove network routes from contianer and update container config.
func DelRoutes(ctr *container.Container, routes []*types.Route, updateConfigOnly bool) error {
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

	var retErr []error
	for _, r := range routes {
		if exist := hConfig.IsRouteExist(r); !exist {
			errinfo := fmt.Sprint("Route(", r, ") is not added by isulad-tools, can not remove it, please check input parameter.")
			retErr = append(retErr, errors.New(errinfo))
			continue
		}
		for _, route := range hConfig.GetRoutes(r) {
			if err := hConfig.UpdateNetworkRoutes(route, false); err != nil {
				retErr = append(retErr, err)
				continue
			}
			// for running container only
			if !updateConfigOnly && ctr.Pid() > 0 && ctr.CheckPidExist() {
				if err := DelRouteFromContainer(ctr.NetNsPath(), route); err != nil {
					// roll back
					hConfig.UpdateNetworkRoutes(route, true)
					retErr = append(retErr, err)
					continue
				}
			}
			fmt.Fprintf(os.Stdout, "Remove route from container %s, route: %s done\n", ctr.Name(), route.String())
			logrus.Infof("Remove route from container %s, route: %s done", ctr.Name(), route.String())
		}
	}
	if len(retErr) == 0 {
		return nil
	}
	for i := 0; i < len(retErr); i++ {
		retErr[i] = fmt.Errorf("%s", retErr[i].Error())
	}
	return errors.New(strings.Trim(fmt.Sprint(retErr), "[]"))
}

// DelRouteFromContainer will add one route to container.
func DelRouteFromContainer(nsPath string, route *types.Route) error {
	var err error
	src := strings.TrimSpace(route.Src)
	dest := strings.TrimSpace(route.Dest)
	gw := strings.TrimSpace(route.Gw)
	dev := strings.TrimSpace(route.Dev)
	if len(src) == 0 && len(gw) == 0 && len(dev) == 0 {
		return fmt.Errorf("src or gw or dev name is required")
	}

	rule := &netlink.Route{}
	if dest == "default" {
		dest = ""
	}
	if len(dest) != 0 {
		rule.Dst, err = netlink.ParseIPNet(dest)
		if err != nil {
			return fmt.Errorf("failed to parse dest %q of route rule", dest)
		}
	}

	if len(src) != 0 {
		rule.Src = net.ParseIP(src)
		if err != nil {
			return fmt.Errorf("failed to parse src ip")
		}
	}

	if len(gw) != 0 {
		rule.Gw = net.ParseIP(gw)
		if err != nil {
			return fmt.Errorf("failed to parse gw ip")
		}
	}

	return nsutils.NsInvoke(nsPath,
		func(nsFD int) error { return nil },
		func(nsFD int) error {
			// executed in container
			if len(dev) != 0 {
				ctrNic, err := netlink.LinkByName(dev)
				if err != nil || ctrNic == nil {
					return fmt.Errorf("failed to get link by name %q: %v", ctrNic, err)
				}
				rule.LinkIndex = ctrNic.Attrs().Index
			}

			if err := netlink.RouteDel(rule); err != nil {
				if strings.Contains(err.Error(), "no such process") {
					return nil
				}
				return fmt.Errorf("failed to remove route: %v", err)
			}
			return nil
		})
}

// ListRoutes will list all filterd network routes in contianer
func ListRoutes(ctr *container.Container, filter *types.Route) ([]*types.Route, error) {
	if err := ctr.Lock(); err != nil {
		return nil, err
	}
	defer ctr.Unlock()

	// create config file handler.
	hConfig, err := hconfig.NewContainerConfig(ctr)
	if err != nil {
		return nil, err
	}
	defer hConfig.Flush()

	return hConfig.GetRoutes(filter), nil
}
