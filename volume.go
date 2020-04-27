// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: path/volume commands
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"isula.org/syscontainer-tools/container"
	"isula.org/syscontainer-tools/libdevice"
	"isula.org/syscontainer-tools/types"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var addPathCommand = cli.Command{
	Name:        "add-path",
	Usage:       "add one or more host paths to container",
	ArgsUsage:   `<container_id> hostpath:containerpath:permission [hostpath:containerpath:permission ...]`,
	Description: `You can add multiple host paths to container.`,
	Flags:       []cli.Flag{},
	Action: func(context *cli.Context) {
		if context.NArg() < 2 {
			fatalf("%s: %q requires a minimum of 2 args", os.Args[0], context.Command.Name)
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		binds, err := getBinds(context, c, true)
		if err != nil {
			fatal(err)
		}

		if err := libdevice.AddPath(c, binds); err != nil {
			fatalf("Failed to add path: %v", err)
		}
		logrus.Infof("add path to container %q successfully", name)
	},
}

var rmPathCommand = cli.Command{
	Name:        "remove-path",
	Usage:       "remove one or more paths from container",
	ArgsUsage:   `<container_id> hostpath:containerpath [hostpath:containerpath ...]`,
	Description: `You can remove multiple host paths from container.`,
	Flags:       []cli.Flag{},
	Action: func(context *cli.Context) {
		if context.NArg() < 2 {
			fatalf("%s: %q requires a minimum of 2 args", os.Args[0], context.Command.Name)
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		binds, err := getBinds(context, c, false)
		if err != nil {
			fatal(err)
		}

		if err := libdevice.RemovePath(c, binds); err != nil {
			fatalf("Failed to remove path: %v", err)
		}
		logrus.Infof("remove path from container %q successfully", name)
	},
}

var listPathCommand = cli.Command{
	Name:      "list-path",
	Usage:     "list all paths mounted to container",
	ArgsUsage: `<container_id>`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "pretty, p",
			Usage: "If this flag is set, list pathes in pretty json form",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept a container-id", os.Args[0], context.Command.Name)
		}
		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		binds, err := libdevice.ListPath(c)
		if err != nil {
			fatalf("Failed to list path in container: %v", err)
		}
		if binds == nil || len(binds) == 0 {
			logrus.Infof("list path in container %q successfully", name)
			return
		}

		bindsData, err := json.Marshal(binds)
		if err != nil {
			fatalf("failed to Marshal path config: %v", err)
		}
		bindsBuffer := new(bytes.Buffer)
		if _, err = bindsBuffer.Write(bindsData); err != nil {
			fatalf("Buffer Write error %v", err)
		}

		if context.Bool("pretty") {
			bindsBuffer.Truncate(0)
			if json.Indent(bindsBuffer, bindsData, "", "\t") != nil {
				fatalf("failed to Indent path data: %v", err)
			}
		}

		if _, err = bindsBuffer.WriteString("\n"); err != nil {
			fatalf("Buffer WriteString error %v", err)
		}
		if _, err := os.Stdout.Write(bindsBuffer.Bytes()); err != nil {
			logrus.Errorf("Write bindsBuffer error %v", err)
		}
		logrus.Infof("list path in container %q successfully", name)
	},
}

func getBinds(context *cli.Context, container *container.Container, create bool) ([]*types.Bind, error) {
	var binds []*types.Bind
	spec := container.GetSpec()
	for k := 1; k < context.NArg(); k++ {
		v := context.Args()[k]
		bind, err := libdevice.ParseBind(v, spec, create)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse bind: %s, %v", v, err)
		}
		binds = append(binds, bind)
	}
	return binds, nil
}
