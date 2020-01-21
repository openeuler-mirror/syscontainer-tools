// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: route commands
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"isula.org/syscontainer-tools/container"
	"isula.org/syscontainer-tools/libnetwork"
	"isula.org/syscontainer-tools/types"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type routeGroup []*types.Route

var addRouteCommand = cli.Command{
	Name:      "add-route",
	Usage:     "add a new network route rule into container",
	ArgsUsage: `<container_id> [{rule1}{rule2}]`,
	Description: `This command is used to add a new route rule into the container,
rule example:
'[{"dest":"default", "gw":"192.168.10.1"},{"dest":"100.10.0.0/16","dev":"eth0","src":"1.1.1.2"}]' .
* dest: dest network, empty means default gateway
* src: route src ip
* gw: route gw
* dev: network device.
`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "update-config-only",
			Usage: "If this flag is set, will not add the route table to container but update config only.",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 2 {
			fatalf("%s: %q must accept container-id and rules", os.Args[0], context.Command.Name)
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		rules := strings.TrimSpace(context.Args()[1])
		if len(rules) == 0 {
			fatalf("rule can't be empty")
		}

		rg := make(routeGroup, 0)
		if err := json.Unmarshal([]byte(rules), &rg); err != nil {
			fatalf("malformed rule format: %v", err)
		}
		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}

		if err := libnetwork.AddRoutes(ctr, rg, context.Bool("update-config-only")); err != nil {
			fatalf("failed to add route: %v", err)
		}
		logrus.Infof("add route to container %q successfully", ctrName)
	},
}

var rmRouteCommand = cli.Command{
	Name:      "remove-route",
	Usage:     "remove a network route rule from container",
	ArgsUsage: `<container_id> [{rule1}{rule2}]`,
	Description: `This command is used to remove route rules from the container,
rule example:
'[{"dest":"default", "gw":"192.168.10.1"},{"dest":"100.10.0.0/16","dev":"eth0","src":"1.1.1.2"}]' .
* dest: dest network, empty means default gateway
* src: route src ip
* gw: route gw
* dev: network device.
`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "update-config-only",
			Usage: "If this flag is set, will not del the route table from container but update config only.",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 2 {
			fatalf("%s: %q must accept container-id and rules", os.Args[0], context.Command.Name)
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		rules := strings.TrimSpace(context.Args()[1])
		if len(rules) == 0 {
			fatalf("rule can't be empty")
		}

		rg := make(routeGroup, 0)
		if err := json.Unmarshal([]byte(rules), &rg); err != nil {
			fatalf("malformed rule format: %v", err)
		}
		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}

		if err := libnetwork.DelRoutes(ctr, rg, context.Bool("update-config-only")); err != nil {
			fatalf("failed to remove route: %v", err)
		}
		logrus.Infof("remove route from container %q successfully", ctrName)
	},
}

var listRouteCommand = cli.Command{
	Name:      "list-route",
	Usage:     "list all filterd network route rules in container",
	ArgsUsage: `<container_id>`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "pretty, p",
			Usage: "If this flag is set, list routes in pretty json form",
		},
		cli.StringFlag{
			Name:  "filter, f",
			Usage: "Filter output based on conditions provided. E.g. '{\"dest\":\"1.1.1.0/24\", \"gw\":\"10.1.1.254\"}'",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept container-id", os.Args[0], context.Command.Name)
		}
		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		ctrName := strings.TrimSpace(context.Args()[0])
		if len(ctrName) == 0 {
			fatalf("container-id can't be empty")
		}

		filterStr := strings.TrimSpace(context.String("filter"))
		if len(filterStr) == 0 {
			filterStr = "{}"
		}

		filter := new(types.Route)
		if err := json.Unmarshal([]byte(filterStr), filter); err != nil {
			fatalf("malformed filter format: %v", err)
		}
		ctr, err := container.New(ctrName)
		if err != nil {
			fatalf("failed to get container info: %v", err)
		}

		routes, err := libnetwork.ListRoutes(ctr, filter)
		if err != nil {
			fatalf("failed to get list routes: %v", err)
		}
		if routes == nil || len(routes) == 0 {
			logrus.Infof("list route in container %q successfully", ctrName)
			return
		}

		routeData, err := json.Marshal(routes)
		if err != nil {
			fatalf("failed to Marshal route config: %v", err)
		}
		routeBuffer := new(bytes.Buffer)
		if _, err = routeBuffer.Write(routeData); err != nil {
			fatalf("Buffer Write error %v", err)
		}

		if context.Bool("pretty") {
			routeBuffer.Truncate(0)
			if json.Indent(routeBuffer, routeData, "", "\t") != nil {
				fatalf("failed to Indent route data: %v", err)
			}
		}

		if _, err = routeBuffer.WriteString("\n"); err != nil {
			fatalf("Buffer WriteString error %v", err)
		}
		if _, err = os.Stdout.Write(routeBuffer.Bytes()); err != nil {
			logrus.Errorf("Write routeBuffer error %v", err)
		}
		logrus.Infof("list route in container %q successfully", ctrName)
	},
}
