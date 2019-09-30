// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: network interface commands
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/urfave/cli"

	"github.com/docker/libnetwork/netutils"
	"github.com/sirupsen/logrus"
	"isula.org/isulad-tools/container"
	"isula.org/isulad-tools/libnetwork"
	"isula.org/isulad-tools/types"
)

var addNicCommand = cli.Command{
	Name:      "add-nic",
	Usage:     "create a new network interface for container",
	ArgsUsage: `<container_id>`,
	Description: `This command is used to create a new network interface in an existing container,
and configure it as you wanted, then attach to specified bridge.
	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Usage: "set network interface type (veth/eth)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "set network interface name: [host:]<container>. for veth type, host could be unset and is random by default. for eth type, host is required",
		},
		cli.StringFlag{
			Name:  "ip",
			Usage: "set ip address. E.g. 172.17.28.2/24",
		},
		cli.StringFlag{
			Name:  "mac",
			Usage: "set mac address. E.g. 00:ff:48:23:e2:bb",
		},
		cli.StringFlag{
			Name:  "bridge",
			Usage: "set bridge name the network interface will attach to, for eth type, bridge cannot be set",
		},
		cli.IntFlag{
			Name:  "mtu",
			Value: 1500,
			Usage: "set mtu",
		},
		cli.IntFlag{
			Name:  "qlen",
			Value: 1000,
			Usage: "set qlen, 1000 by default",
		},
		cli.BoolFlag{
			Name:  "update-config-only",
			Usage: "If this flag is set, will not add network interface to container but update config only",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept a container-id", os.Args[0], context.Command.Name)
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		hostNicName, ctrNicName, err := parseNicName(context.String("name"))
		if err != nil {
			fatalf("failed to parse network name: %s", context.String("name"))
		}

		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}
		if ctrNicName == "" {
			fatalf("failed to get container nic name")
		}

		nicConf := &types.InterfaceConf{
			IP:          context.String("ip"),
			Mac:         context.String("mac"),
			Mtu:         context.Int("mtu"),
			Type:        context.String("type"),
			Bridge:      context.String("bridge"),
			Qlen:        context.Int("qlen"),
			CtrNicName:  ctrNicName,
			HostNicName: hostNicName,
		}

		if err := types.ValidNetworkConfig(nicConf); err != nil {
			fatalf("invalid network option: %v", err)
		}

		if nicConf.HostNicName == "" {
			nicConf.HostNicName = nicConf.CtrNicName
			// interface name length must be less than 15
			if len(nicConf.CtrNicName) > 9 {
				nicConf.HostNicName = nicConf.CtrNicName[:9]
			}
			nicConf.HostNicName, err = netutils.GenerateRandomName(nicConf.HostNicName+"_", 5)
			if err != nil {
				fatalf("failed set host nic name %v", err)
			}
		}
		if err := libnetwork.AddNic(ctr, nicConf, context.Bool("update-config-only")); err != nil {
			fatalf("failed to add nic into container: %v", err)
		}
		logrus.Infof("add network interface to container %s successfully", ctrName)
	},
}

var rmNicCommand = cli.Command{
	Name:      "remove-nic",
	Usage:     "remove a network interface from container",
	ArgsUsage: `<container_id>`,
	Description: `This command is used to remove a network interface from an existing container.
	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Usage: "set network interface type (veth/eth)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "set network interface name: [host:]<container>",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept a container-id", os.Args[0], context.Command.Name)
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		hostNicName, ctrNicName, err := parseNicName(context.String("name"))
		if err != nil {
			fatalf("failed to parse network name: %s", context.String("name"))
		}

		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}

		nicConf := &types.InterfaceConf{
			CtrNicName:  ctrNicName,
			HostNicName: hostNicName,
			Type:        context.String("type"),
		}

		if err := libnetwork.DelNic(ctr, nicConf); err != nil {
			fatalf("failed to remove nic from container: %v", err)
		}
		logrus.Infof("remove network interface from container %v successfully", ctrName)
	},
}

func parseNicName(name string) (string, string, error) {
	names := strings.SplitN(name, ":", 2)
	if len(names) == 1 {
		return "", name, nil
	}

	return names[0], names[1], nil
}

var updateNicCommand = cli.Command{
	Name:      "update-nic",
	Usage:     "update network interfaces in a container",
	ArgsUsage: `<container_id>`,
	Description: `This command is used to update network interfaces in an existing container.
	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "network interface name in container will be updated, must be required",
		},
		cli.StringFlag{
			Name:  "ip",
			Usage: "set ip address. E.g. 172.17.28.2/24",
		},
		cli.StringFlag{
			Name:  "mac",
			Usage: "set mac address. E.g. 00:ff:48:23:e2:bb",
		},
		cli.StringFlag{
			Name:  "bridge",
			Usage: "set bridge name the network interface will attach to",
		},
		cli.IntFlag{
			Name:  "mtu",
			Usage: "set mtu. 0 means keeping old value",
		},
		cli.IntFlag{
			Name:  "qlen",
			Usage: "set qlen, works only on veth type. 0 means keeping old value",
		},
		cli.BoolFlag{
			Name:  "update-config-only",
			Usage: "If this flag is set, will not add network interface to container but update config only",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept a container-id", os.Args[0], context.Command.Name)
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}

		ctrNicName := context.String("name")
		if ctrNicName == "" {
			fatalf("Network interface name in container must be provided")
		}

		// we use qlen < 0 to tell qlen is not set by user, so set it to -1 here
		var qlen int
		if !context.IsSet("qlen") {
			qlen = -1
		} else {
			qlen = context.Int("qlen")
		}

		nicConf := &types.InterfaceConf{
			IP:         context.String("ip"),
			Mac:        context.String("mac"),
			Mtu:        context.Int("mtu"),
			Bridge:     context.String("bridge"),
			Qlen:       qlen,
			CtrNicName: ctrNicName,
		}

		if err := libnetwork.UpdateNic(ctr, nicConf, context.Bool("update-config-only")); err != nil {
			fatalf("failed to upadte nic in container: %v", err)
		}

		logrus.Infof("update network interface in container %v successfully", ctrName)
	},
}

var listNicCommand = cli.Command{
	Name:      "list-nic",
	Usage:     "list all network interfaces in a container",
	ArgsUsage: `<container_id>`,
	Description: `This command is used to list all network interfaces in an existing container.
	`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "pretty, p",
			Usage: "If this flag is set, list nics in pretty json form",
		},
		cli.StringFlag{
			Name:  "filter, f",
			Usage: "Filter output based on conditions provided. E.g. '{\"ip\":\"1.2.3.4/24\", \"Mtu\":1500}'",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept a container-id", os.Args[0], context.Command.Name)
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}

		filterString := context.String("filter")
		if filterString == "" {
			filterString = "{}"
		}
		var filter = new(types.InterfaceConf)
		if err = json.Unmarshal([]byte(filterString), filter); err != nil {
			fatalf("failed to parse filter: %v", err)
		}

		nics, err := libnetwork.ListNic(ctr, filter)
		if err != nil {
			fatalf("failed to list nic in container: %v", err)
		}
		if nics == nil || len(nics) == 0 {
			logrus.Infof("list network interface in container %q successfully", ctrName)
			return
		}

		nicData, err := json.Marshal(nics)
		if err != nil {
			fatalf("failed to Marshal nic config: %v", err)
		}
		nicBuffer := new(bytes.Buffer)
		if _, err = nicBuffer.Write(nicData); err != nil {
			fatalf("Buffer Write error %v", err)
		}

		if context.Bool("pretty") {
			nicBuffer.Truncate(0)
			if json.Indent(nicBuffer, nicData, "", "\t") != nil {
				fatalf("failed to Indent nic data: %v", err)
			}
		}
		if _, err = nicBuffer.WriteString("\n"); err != nil {
			fatalf("Buffer WriteString error %v", err)
		}
		if _, err = os.Stdout.Write(nicBuffer.Bytes()); err != nil {
			logrus.Errorf("Write nicBuffer.Bytes error: %v", err)
		}
		logrus.Infof("list network interface in container %v successfully", ctrName)
	},
}
