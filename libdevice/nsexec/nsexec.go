// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: ns exec in container namespace
// Author: zhangwei
// Create: 2018-01-18

package nsexec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"isula.org/syscontainer-tools/types"
	"isula.org/syscontainer-tools/utils"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/vishvananda/netlink/nl"
)

const (
	// AddDeviceMsg is a parent and child process message type, for adding device operation
	AddDeviceMsg = 1
	// RemoveDeviceMsg is a parent and child process message type, for removing device operation
	RemoveDeviceMsg = 2
	// AddBindMsg is a parent and child process message type, for adding bind operation
	AddBindMsg = 3
	// RemoveBindMsg is a parent and child process message type, for removing bind operation
	RemoveBindMsg = 4
	// AddTransferBaseMsg is a parent and child process message type, for adding sharing
	AddTransferBaseMsg = 5
	// UpdateSysctlMsg is a parent and child process message type, for updateing sysctl
	UpdateSysctlMsg = 6
	// MountMsg is a parent and child process message type, for remount /dev/ to remove nodev
	MountMsg = 7
	// InitPipe is a parent and child process env name, used to pass the init pipe number to child process
	InitPipe = "_LIBCONTAINER_INITPIPE"
	// WorkType is a parent and child process env name, used to pass the work type to child process
	WorkType = "_ISULAD_TOOLS_WORKTYPE"
	// NsEnterReexecName is the reexec name, see reexec package
	NsEnterReexecName = "nsenter-init"
)

type nsexecDriver struct {
}

type pid struct {
	Pid int `json:"Pid"`
}

// NewNSExecDriver creates the nsexecDriver
func NewNSExecDriver() NsDriver {
	return &nsexecDriver{}
}

func (ns *nsexecDriver) exec(nsPaths string, worktype int, data interface{}) error {
	parent, child, err := utils.NewPipe()
	if err != nil {
		return err
	}
	cmd := &exec.Cmd{
		Path:       "/proc/self/exe",
		Args:       []string{NsEnterReexecName},
		ExtraFiles: []*os.File{child},
		Env: []string{fmt.Sprintf("%s=3", InitPipe),
			fmt.Sprintf("%s=%d", WorkType, worktype)},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	r := nl.NewNetlinkRequest(int(libcontainer.InitMsg), 0)

	r.AddData(&libcontainer.Bytemsg{
		Type:  libcontainer.NsPathsAttr,
		Value: []byte(nsPaths),
	})

	// send nspath to child process through _ISULAD_TOOLS_INITPIPE, to join container ns.
	if _, err := io.Copy(parent, bytes.NewReader(r.Serialize())); err != nil {
		return err
	}
	// send the config to child
	if err := utils.WriteJSON(parent, data); err != nil {
		return err
	}

	// wait for command
	if err := cmd.Wait(); err != nil {
		return err
	}

	decoder := json.NewDecoder(parent)
	var pid *pid
	if err := decoder.Decode(&pid); err != nil {
		fmt.Fprintf(os.Stderr, "fail to decode pid:%v, but it may not affect later process", err)
	}

	// read error message
	var msg types.ErrMsg
	if err := decoder.Decode(&msg); err != nil {
		return err
	}

	if msg.Error != "" {
		return fmt.Errorf("%s", msg.Error)
	}
	return nil
}

// AddDevice is a low level function which implements how to add devices to a container.
func (ns *nsexecDriver) AddDevice(pid string, device *types.Device, force bool) error {
	namespaces := []string{"mnt"}
	nsPaths := buildNSString(pid, namespaces)

	msg := &types.AddDeviceMsg{
		Force:  force,
		Device: device,
	}

	return ns.exec(nsPaths, AddDeviceMsg, msg)
}

// RemoveDevice is a low level function which implements how to remove devices from a container.
func (ns *nsexecDriver) RemoveDevice(pid string, device *types.Device) error {
	namespaces := []string{"mnt"}
	nsPaths := buildNSString(pid, namespaces)

	return ns.exec(nsPaths, RemoveDeviceMsg, device)
}

// AddTransferBase adds transfer path between container and host for sharing files
func (ns *nsexecDriver) AddTransferBase(pid string, bind *types.Bind) error {
	namespaces := []string{"mnt"}
	nsPaths := buildNSString(pid, namespaces)

	return ns.exec(nsPaths, AddTransferBaseMsg, bind)
}

// AddBind is a low level function which implements how to add binds to a container.
func (ns *nsexecDriver) AddBind(pid string, bind *types.Bind) error {
	namespaces := []string{"mnt"}
	nsPaths := buildNSString(pid, namespaces)

	return ns.exec(nsPaths, AddBindMsg, bind)
}

// RemoveBind is a low level function which implements how to remove binds from a container.
func (ns *nsexecDriver) RemoveBind(pid string, bind *types.Bind) error {
	namespaces := []string{"mnt"}
	nsPaths := buildNSString(pid, namespaces)

	return ns.exec(nsPaths, RemoveBindMsg, bind)
}

// UpdateSysctl is a low level function which implements how to update sysctl for a userns enabled contianer
func (ns *nsexecDriver) UpdateSysctl(pid string, sysctl *types.Sysctl) error {
	namespaces := []string{"ipc", "net", "mnt"}
	nsPaths := buildNSString(pid, namespaces)

	return ns.exec(nsPaths, UpdateSysctlMsg, sysctl)
}

func (ns *nsexecDriver) Mount(pid string, mount *types.Mount) error {
	namespaces := []string{"mnt"}
	nsPaths := buildNSString(pid, namespaces)

	return ns.exec(nsPaths, MountMsg, mount)
}

func buildNSString(pid string, namespaces []string) string {
	var nsPaths string
	for _, ns := range namespaces {
		if nsPaths != "" {
			nsPaths += ","
		}
		nsPaths += fmt.Sprintf("%s:/proc/%s/ns/%s", ns, pid, ns)
	}
	return nsPaths
}
