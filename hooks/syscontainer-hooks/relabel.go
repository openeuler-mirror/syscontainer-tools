// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: selinux relabel operation
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/opencontainers/runc/libcontainer/selinux"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink/nl"
	hconfig "isula.org/syscontainer-tools/config"
	"isula.org/syscontainer-tools/libdevice/nsexec"
	"isula.org/syscontainer-tools/utils"
)

var (
	autoRelabel            = "/.autorelabel"
	autoRelabelInContainer = "/.autorelabel_in_container"
	containerautoRelabel   = "/.container_autorelabel"
	relabelBin             = "/usr/bin/autorelabel_container"
	systemdServiceFile     = "/etc/systemd/system/multi-user.target.wants/autorelabel.service"
	upstartServiceFile     = "/etc/init/autorelabel.conf"
	systemdInit            = "systemd"
	appName                = "oci-relabel-hook"
	usage                  = "oci-relabel-hook poststart|poststop"
	relabelRexec           = "reexec-relabel"

	autoRelabelService = `#!/bin/bash
. /etc/selinux/config
setenforce 0
semodule -R
if [ -f "%s" ]; then
	restorecon -R /
	rm -rf %s
	reboot -f
else
	if [ "$SELINUX" = "enforcing" ]; then
		setenforce 1
	else
		setenforce 0
	fi
fi`

	upstartService = `start on startup
task
console output
script
	logger "upstart-autorelabel start"
	exec %s
end script`

	systemdService = `[Unit]
Description=Relabel all container's filesystems, if necessary
DefaultDependencies=no
Requires=local-fs.target
Conflicts=shutdown.target
After=local-fs.target
Before=sysinit.target shutdown.target

[Service]
ExecStart=%s

[Install]
WantedBy=multi-user.target

`
)

func init() {
	reexec.Register(relabelRexec, RelabelInMntNs)
}

// RelabelInMntNs relabel in container mount namespace
func RelabelInMntNs() {
	var s configs.HookState
	if err := json.NewDecoder(os.Stdin).Decode(&s); err != nil {
		logrus.Errorf("[oci relabel] Failed to decode stdin: %v", err)
		return
	}

	if err := preStartNs(&s); err != nil {
		logrus.Errorf("[oci relabel] Failed to relabel in mnt ns: %v", err)
	}
}

func relabelSystemd(rootfs string) error {
	logrus.Info("systemd autorelable")
	autoRel := fmt.Sprintf(autoRelabelService, autoRelabelInContainer, autoRelabelInContainer)
	if err := ioutil.WriteFile(filepath.Join(rootfs, relabelBin), []byte(autoRel), 0700); err != nil {
		return err
	}
	systemdService := fmt.Sprintf(systemdService, relabelBin)
	if err := ioutil.WriteFile(filepath.Join(rootfs, systemdServiceFile), []byte(systemdService), 0600); err != nil {
		return err
	}

	return nil
}

func relabelUpstart(rootfs string) error {
	logrus.Info("upstart autorelable")
	autoRel := fmt.Sprintf(autoRelabelService, autoRelabelInContainer, autoRelabelInContainer)
	if err := ioutil.WriteFile(filepath.Join(rootfs, relabelBin), []byte(autoRel), 0700); err != nil {
		return err
	}

	upstartService := fmt.Sprintf(upstartService, relabelBin)
	if err := ioutil.WriteFile(filepath.Join(rootfs, upstartServiceFile), []byte(upstartService), 0600); err != nil {
		return err
	}
	return nil
}

func relabel(rootfs string) error {
	if utils.IsSystemdInit(rootfs) {
		return relabelSystemd(rootfs)
	}
	return relabelUpstart(rootfs)
}

func preStartNs(s *configs.HookState) error {
	var (
		se                string
		err               error
		attr              string
		seconfig          = "/etc/selinux/config"
		seconfigContainer = s.Root + "/etc/selinux/config"
	)

	if se, err = utils.SeconfigGet(seconfig, "SELINUX"); err != nil {
		return err
	}
	// don't exec hook's function if SELinux is disabled in host
	if se == "disabled" {
		logrus.Infof("Host SELinux disabled")
		return nil
	}

	// set permissive to host /etc/selinux/config
	if err = utils.SeconfigSet(seconfig, "SELINUX", "permissive"); err != nil {
		return err
	}

	if se, err = utils.SeconfigGet(seconfigContainer, "SELINUX"); err != nil {
		return err
	}
	// proposal from it
	// don't exec hook's function if SELinux is disabled in container
	if se == "disabled" {
		logrus.Infof("Container SELinux disabled")
		return nil
	}

	// mount selinuxfs
	if err = syscall.Mount("none", s.Root+utils.GetSelinuxMountPount(s.Root), "selinuxfs", 0, ""); err != nil {
		return err
	}
	// start a container in the first time, it need relabel, so create a /.autorelabel file
	if !utils.IsExist(s.Root + containerautoRelabel) {
		if err := ioutil.WriteFile(filepath.Join(s.Root, autoRelabel), []byte(""), 0600); err != nil {
			logrus.Errorf("WriteFile err: %v", err)
		}
		if err := ioutil.WriteFile(filepath.Join(s.Root, containerautoRelabel), []byte(""), 0600); err != nil {
			logrus.Errorf("WriteFile err: %v", err)
		}
	}

	// relabel container' rootfs, just create a systemd service file, and the relabel process is executed in container.
	if err = relabel(s.Root); err != nil {
		return err
	}

	// make sure relabelBin can execute exactly
	hostRelabelBin := filepath.Join(s.Root, relabelBin)
	if attr, err = selinux.Getfilecon(hostRelabelBin); err != nil {
		logrus.Errorf("Getfilecon %s err", hostRelabelBin)
		return nil
	}
	con := utils.NewContext(attr)
	con.SetType("init_exec_t")
	selinux.Setfilecon(hostRelabelBin, con.Get())
	logrus.Infof("%s [%s]", hostRelabelBin, con.Get())

	if utils.IsExist(s.Root + autoRelabel) {
		if err := ioutil.WriteFile(filepath.Join(s.Root, autoRelabelInContainer), []byte(""), 0600); err != nil {
			logrus.Errorf("WriteFile err: %v", err)
		}
		if err := syscall.Unlink(filepath.Join(s.Root, autoRelabel)); err != nil {
			return err
		}
	}
	return nil
}

func preStartClone(s *configs.HookState) error {
	parent, child, err := utils.NewPipe()
	if err != nil {
		return nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=3", nsexec.InitPipe))
	cmd := &exec.Cmd{
		Path:       "/proc/self/exe",
		Args:       []string{relabelRexec},
		ExtraFiles: []*os.File{child},
		Env:        env,
		Stdin:      bytes.NewReader(b),
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	namespaces := []string{
		fmt.Sprintf("mnt:/proc/%d/ns/mnt", s.Pid),
	}
	r := nl.NewNetlinkRequest(int(libcontainer.InitMsg), 0)
	r.AddData(&libcontainer.Bytemsg{
		Type:  libcontainer.NsPathsAttr,
		Value: []byte(strings.Join(namespaces, ",")),
	})
	if _, err := io.Copy(parent, bytes.NewReader(r.Serialize())); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// PrestartRelabel handles oci relabel for prestart state
func PrestartRelabel(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	if err := preStartClone(state); err != nil {
		return err
	}
	return nil
}

// PostStopRelabel handles oci relabel for post-stop state
func PostStopRelabel(state *configs.HookState, hookConfig *hconfig.ContainerHookConfig, spec *specs.Spec) error {
	if utils.IsSystemdInit(state.Root) {
		if err := syscall.Unlink(filepath.Join(state.Root + systemdServiceFile)); err != nil {
			logrus.Errorf("syscall.Unlink state.Root err: %v", err)
		}
	} else {
		if err := syscall.Unlink(filepath.Join(state.Root + upstartServiceFile)); err != nil {
			logrus.Errorf("syscall.Unlink not state.Root err: %v", err)
		}
	}
	return nil
}
