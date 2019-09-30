// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: main funtion
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"isula.org/isulad-tools/config"
	"isula.org/isulad-tools/utils"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	gitCommit = ""
	version   = ""
)

const (
	usage     = `Enhanced tools for isulad`
	syslogTag = "tools "
)

// fatal prints the error's details
// then exits the program with an exit status of 1.
func fatal(err error) {
	// make sure the error is written to the logger
	logrus.Error(err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fatalf(t string, v ...interface{}) {
	fatal(fmt.Errorf(t, v...))
}

func mainWork() {
	app := cli.NewApp()
	app.Name = "isulad-tools"
	app.Usage = usage
	v := []string{
		version,
	}
	if gitCommit != "" {
		v = append(v, fmt.Sprintf("commit: %s", gitCommit))
	}
	app.Version = strings.Join(v, "\n")
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log",
			Value: "/dev/null",
			Usage: "set the log file path('.' stands for stdout)",
		},
		cli.StringFlag{
			Name:  "log-level",
			Value: "info",
			Usage: "set the level for logging(debug, info, warn, error, fatal, panic)",
		},
		cli.StringFlag{
			Name:  "log-format",
			Value: "text",
			Usage: "set the format used by logs ('text' or 'json')",
		},
		cli.StringFlag{
			Name:  "syslog-service",
			Value: "unix:///dev/log",
			Usage: "set syslog service",
		},
	}

	app.Commands = []cli.Command{
		addDevCommand,
		addNicCommand,
		addPathCommand,
		addRouteCommand,
		relabelCommand,
		rmDevCommand,
		rmNicCommand,
		rmPathCommand,
		rmRouteCommand,
		listNicCommand,
		listPathCommand,
		listRouteCommand,
		listDevCommand,
		updateDevCommand,
		updateNicCommand,
	}

	app.CommandNotFound = func(context *cli.Context, command string) {
		fatalf("unknown subcommand %v", command)
	}

	app.Before = func(context *cli.Context) error {
		if err := os.MkdirAll(config.IsuladToolsDir, 0666); err != nil {
			logrus.Errorf("failed to set isulad-tools dir: %v", err)
		}

		if logpath := context.GlobalString("log"); logpath != "" {
			var logfile *os.File
			var err error
			if logpath == "." {
				logfile = os.Stdout
			} else {
				logfile, err = os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0600)
				if err != nil {
					return err
				}
			}
			logrus.SetOutput(logfile)
		}

		lvl, err := logrus.ParseLevel(context.GlobalString("log-level"))
		if err != nil {
			logrus.Fatalf("unknown log-level %v", err)
		}
		logrus.SetLevel(lvl)

		switch context.GlobalString("log-format") {
		case "text":
			logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})
		case "json":
			logrus.SetFormatter(new(logrus.JSONFormatter))
		default:
			fatalf("unknown log-format %s", context.GlobalString("log-format"))
		}

		if err := utils.HookSyslog(context.GlobalString("syslog-service"), syslogTag); err != nil {
			logrus.Errorf("failed to set syslog: %v", err)
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fatal(err)
	}
}
func main() {
	if reexec.Init() {
		// `reexec routine` was registered in isulad-tools/libdevice
		// Sub nsenter process will come here.
		// Isulad reexec package do not handle errors.
		// And sub isulad-tools nsenter init process will send back the error message to parenet through pipe.
		// So here do not need to handle errors.
		return
	}
	signal.Ignore(syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	mainWork()
}
