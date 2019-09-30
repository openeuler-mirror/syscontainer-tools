// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: selinux relabel commands
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"isula.org/isulad-tools/utils"

	"github.com/opencontainers/runc/libcontainer/selinux"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	// Seconfig is absolute path for SELinux config file
	Seconfig    = "/etc/selinux/config"
	hostSystemd = "/lib/systemd/systemd"
)

type selinuxCommand struct {
	cmd  string
	argv []string
}

type selinuxContext struct {
	conType string
	path    string
}

func relabelIsuladBinary(path, bin string) error {
	cmd := exec.Command("semanage", []string{"fcontext", "-a", "-t", "init_exec_t", fmt.Sprintf("%s/%s", path, bin)}...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func restartIsulad() error {
	restart := selinuxCommand{"systemctl", []string{"restart", "lcrd"}}
	cmd := exec.Command(restart.cmd, restart.argv...)
	logrus.Infof("%s %v", restart.cmd, restart.argv)
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%s %v: %v", restart.cmd, restart.argv, err)
		return err
	}
	return nil
}

func relabel(path string) error {
	var seType string
	var attr string
	var err error

	if seType, err = utils.SeconfigGet(Seconfig, "SELINUXTYPE"); err != nil {
		return err
	}
	utils.SeconfigSet(Seconfig, "SELINUX", "permissive")
	fileContexts := fmt.Sprintf("/etc/selinux/%s/contexts/files/file_contexts.local", seType)
	if !utils.IsExist(fileContexts) {
		if err = ioutil.WriteFile(fileContexts, []byte(""), 0600); err != nil {
			logrus.Errorf("ioutil.WriteFile err: %s", err)
		}
	}

	preStarts := []selinuxCommand{
		{"setenforce", []string{"0"}},
	}

	for _, prestart := range preStarts {
		cmd := exec.Command(prestart.cmd, prestart.argv...)
		logrus.Infof("%s %v", prestart.cmd, prestart.argv)
		if err := cmd.Run(); err != nil {
			logrus.Errorf("%s %v: %v", prestart.cmd, prestart.argv, err)
			return err
		}
	}

	if attr, err = selinux.Getfilecon(hostSystemd); err != nil {
		return nil
	}
	modifyContexts := []selinuxContext{
		{"init_exec_t", path + "/lcrd"},
	}
	con := utils.NewContext(attr)
	for _, context := range modifyContexts {
		con.SetType(context.conType)
		logrus.Infof("%s [%s]", context.path, con.Get())
		selinux.Setfilecon(context.path, con.Get())
	}

	return restartIsulad()
}

func setUpSelinuxLabel(path, rootfs string) error {
	if err := relabel(path); err != nil {
		return err
	}
	return nil
}

var relabelCommand = cli.Command{
	Name:        "relabel",
	Usage:       "relabel rootfs for running SELinux in system container",
	ArgsUsage:   `[--isulad-path path] [--rootfs rootfs]`,
	Description: `relabel rootfs for running SELinux in system container(a systemd based os is required)`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "isulad-path",
			Value: "/usr/bin",
			Usage: "isulad's install path",
		},
		cli.StringFlag{
			Name:  "rootfs",
			Usage: "the absolute path for isulad's rootfs",
		},
	},
	Action: func(context *cli.Context) {
		if err := setUpSelinuxLabel(context.String("isulad-path"), context.String("rootfs")); err != nil {
			fatal(err)
		}
		// print result to stdout
		fmt.Printf("SELinux relabel success\n")
		logrus.Infof("SELinux relabel successfully")
	},
}
