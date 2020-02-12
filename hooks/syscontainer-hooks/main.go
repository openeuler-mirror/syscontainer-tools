// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: syscontainer hook main function
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/opencontainers/runc/libcontainer/configs"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	hconfig "isula.org/syscontainer-tools/config"
	"isula.org/syscontainer-tools/container"
	"isula.org/syscontainer-tools/utils"
)

var (
	defaultHookConfigFile = "device_hook.json"
	syslogTag             = "hook "
	bundleConfigFile      = "config.json"
)

type hookData struct {
	spec        *specs.Spec
	state       *configs.HookState
	hookConfig  *hconfig.ContainerHookConfig
	storagePath string
}

func setupLog(logfile string) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0600)
		if err != nil {
			return
		}
		logrus.SetOutput(f)
		return
	}
	fm := &logrus.TextFormatter{DisableTimestamp: true, DisableColors: true}
	logrus.SetFormatter(fm)
	if err := utils.HookSyslog("", syslogTag); err != nil {
		fmt.Fprintf(os.Stdout, "%v", err)
	}
}

func fatal(err error) {
	if err != nil {
		logrus.Error(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepareHookData() (*hookData, error) {
	var (
		err                  error
		spec                 *specs.Spec
		compatSpec           *specs.CompatSpec
		state                *configs.HookState
		hookConfig           *hconfig.ContainerHookConfig
		containerStoragePath = ""
	)

	if state, err = utils.ParseHookState(os.Stdin); err != nil {
		logrus.Errorf("Parse Hook State Failed: %v", err)
		return nil, err
	}

	// Load container OCI spec from config
	configFile := bundleConfigFile
	if spec, err = utils.LoadSpec(filepath.Join(state.Bundle, configFile)); err != nil {
		if compatSpec, err = utils.LoadCompatSpec(filepath.Join(state.Bundle, configFile)); err != nil {
			logrus.Errorf("Failed to load spec for contianer %s: %v", state.ID, err)
			return nil, err
		}
	}

	if compatSpec != nil {
		capabilities := compatSpec.Process.Capabilities
		spec = &compatSpec.Spec
		spec.Process = compatSpec.Process.Process
		spec.Process.Capabilities = &specs.LinuxCapabilities{
			Bounding:    capabilities,
			Effective:   capabilities,
			Inheritable: capabilities,
			Permitted:   capabilities,
			Ambient:     capabilities,
		}
	}

	if containerStoragePath, err = utils.GetContainerStoragePath(); err != nil {
		logrus.Errorf("Failed to get container storage path: %v", err)
		return nil, err
	}

	configPath := filepath.Join(containerStoragePath, state.ID, defaultHookConfigFile)
	if _, err := os.Stat(configPath); err != nil {
		// not an error, user do not add/remove device to/from this container.
		return &hookData{
			spec:        spec,
			state:       state,
			hookConfig:  &hconfig.ContainerHookConfig{},
			storagePath: containerStoragePath,
		}, nil
	}

	// Load devices and binds config for container.
	if hookConfig, err = hconfig.LoadContainerHookConfig(configPath); err != nil {
		logrus.Errorf("Failed to parse Config File for container %s: %v", state.ID, err)
		return nil, err
	}

	return &hookData{
		spec:        spec,
		state:       state,
		hookConfig:  hookConfig,
		storagePath: containerStoragePath,
	}, nil
}

func main() {
	if reexec.Init() {
		// `reexec routine` was registered in syscontainer-tools/libdevice
		// Sub nsenter process will come here.
		// Isulad reexec package do not handle errors.
		// And sub device-hook nsenter init process will send back the error message to parenet through pipe.
		// So here do not need to handle errors.
		return
	}
	signal.Ignore(syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	flLogfile := flag.String("log", "", "set output log file")
	flMode := flag.String("state", "", "set syscontainer hook state mode: prestart or poststop")
	// No requirements at present, by default don't enable this function.
	flWithRelabel := flag.Bool("with-relabel", false, "syscontainer hook enable oci relabel hook function")

	flag.Parse()

	setupLog(*flLogfile)
	hData, err := prepareHookData()
	if err != nil {
		return
	}

	if err := os.MkdirAll(hconfig.IsuladToolsDir, 0666); err != nil {
		logrus.Errorf("failed to set syscontainer-tools dir: %v", err)
	}

	switch *flMode {
	case "prestart":
		if hData.state.Pid <= 0 {
			logrus.Errorf("can't get correct pid of container: %d", hData.state.Pid)
			return
		}
		if err := prestartHook(hData, *flWithRelabel); err != nil {
			fatal(err)
		}

		if err := updateHookData(hData); err != nil {
			fatal(err)
		}
	case "poststart":
		if hData.state.Pid <= 0 {
			logrus.Errorf("can't get correct pid of container: %d", hData.state.Pid)
			return
		}
		if err := poststartHook(hData, *flWithRelabel); err != nil {
			fatal(err)
		}

		if err := updateHookData(hData); err != nil {
			fatal(err)
		}

	case "poststop":
		postStopHook(hData, *flWithRelabel)
	}
}

func updateHookData(data *hookData) error {

	var (
		err                  error
		containerStoragePath = ""
	)

	c := &container.Container{}
	if data.storagePath != "" {
		containerStoragePath = data.storagePath
	} else {
		containerStoragePath, err = utils.GetContainerStoragePath()
		if err != nil {
			logrus.Errorf("Failed to get container storage path: %v", err)
			return err
		}
	}

	c.SetContainerPath(filepath.Join(containerStoragePath, data.state.ID))

	if err := c.Lock(); err != nil {
		return err
	}
	defer c.Unlock()

	defer func() {
		if err := data.hookConfig.Flush(); err != nil {
			logrus.Infof("config Flush error:%v", err)
		}
	}()
	return nil
}
