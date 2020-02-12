// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: bind and device operation in container namespace
// Author: zhangwei
// Create: 2018-01-18

package libdevice

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"isula.org/syscontainer-tools/libdevice/nsexec"
	"isula.org/syscontainer-tools/pkg/mount"
	"isula.org/syscontainer-tools/types"
	"isula.org/syscontainer-tools/utils"
)

func init() {
	reexec.Register(nsexec.NsEnterReexecName, WorkInContainer)
}

func setupPipe(name string) (*os.File, error) {
	v := os.Getenv(name)

	fd, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf("unable to convert %s=%s to int", name, v)
	}
	return os.NewFile(uintptr(fd), "pipe"), nil
}

func setupWorkType(name string) (int, error) {
	v := os.Getenv(name)

	worktype, err := strconv.Atoi(v)
	if err != nil {
		return -1, fmt.Errorf("unable to convert %s=%s to int", name, v)
	}
	return worktype, nil
}

// WorkInContainer will handle command in new namespace(container).
func WorkInContainer() {
	var err error
	var worktype int
	var pipe *os.File
	pipe, err = setupPipe(nsexec.InitPipe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return
	}

	// when pipe setup, should always send back the errors.
	defer func() {
		var msg types.ErrMsg
		if err != nil {
			msg.Error = fmt.Sprintf("%s", err.Error())
		}
		if err := utils.WriteJSON(pipe, msg); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}()

	worktype, err = setupWorkType(nsexec.WorkType)
	if err != nil {
		return
	}

	// handle work here:
	switch worktype {
	case nsexec.AddDeviceMsg:
		err = doAddDevice(pipe)
	case nsexec.RemoveDeviceMsg:
		err = doRemoveDevice(pipe)
	case nsexec.AddBindMsg:
		err = doAddBind(pipe)
	case nsexec.RemoveBindMsg:
		err = doRemoveBind(pipe)
	case nsexec.AddTransferBaseMsg:
		err = doAddTransferBase(pipe)
	case nsexec.UpdateSysctlMsg:
		err = doUpdateSysctl(pipe)
	case nsexec.MountMsg:
		err = doMount(pipe)
	default:
		err = fmt.Errorf("unkown worktype=(%d)", worktype)
	}
	// do not need to check err here because we have check in defer
	return
}

// writeSystemProperty writes the value to a path under /proc/sys as determined from the key.
// For e.g. net.ipv4.ip_forward translated to /proc/sys/net/ipv4/ip_forward.
func writeSystemProperty(key, value string) error {
	keyPath := strings.Replace(key, ".", "/", -1)
	return ioutil.WriteFile(path.Join("/proc/sys", keyPath), []byte(value), 0644)
}

func doUpdateSysctl(pipe *os.File) error {
	var sysctl types.Sysctl
	if err := json.NewDecoder(pipe).Decode(&sysctl); err != nil {
		return err
	}
	return writeSystemProperty(sysctl.Key, sysctl.Value)
}

func doMount(pipe *os.File) error {
	var mnt types.Mount
	if err := json.NewDecoder(pipe).Decode(&mnt); err != nil {
		return err
	}

	if mnt.Type == "move" {
		_, err := os.Stat(mnt.Destination)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("doMount: stat %s in container failed, err: %s", mnt.Destination, err)
		}
		if err := os.MkdirAll(mnt.Destination, 0600); err != nil {
			return fmt.Errorf("doMount: create mount destination in container failed, err: %s", err)
		}
		return syscall.Mount(mnt.Source, mnt.Destination, "", syscall.MS_MOVE, "")
	}
	if mnt.Type == "bind" {
		_, err := os.Stat(mnt.Destination)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("doMount: stat %s in container failed, err: %s", mnt.Destination, err)
		}
		fi, err := os.Stat(mnt.Source)
		if err != nil {
			return fmt.Errorf("doMount: stat %s in container failed, err: %s", mnt.Source, err)
		}
		if err := os.Chown(mnt.Source, mnt.UID, mnt.GID); err != nil {
			return fmt.Errorf("chown changes the numeric uid and gid of the name file, err: %s", err)
		}
		if fi.Mode().IsDir() {
			if err := os.MkdirAll(mnt.Destination, 0600); err != nil {
				return fmt.Errorf("doMount: create mount destination in container failed, err: %s", err)
			}
		} else {
			f, err := os.OpenFile(mnt.Destination, os.O_RDWR|os.O_CREATE, 0600)
			if err != nil {
				return fmt.Errorf("fail to create %s, err: %s", mnt.Destination, err)
			}
			f.Close()
		}
		return mount.Mount(mnt.Source, mnt.Destination, "none", mnt.Options)
	}
	if mnt.Type == "link" {
		if err := os.Symlink(mnt.Source, mnt.Destination); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("doMount: symlink %s %s %s", mnt.Source, mnt.Destination, err)
		}
		if err := os.Lchown(mnt.Destination, mnt.UID, mnt.GID); err != nil {
			logrus.Errorf("os.Lchown error: %s", err)
		}
		if err := os.Remove(strings.Replace(mnt.Destination, "/dev", "/.dev", -1)); err != nil {
			logrus.Errorf("os.Remove error: %s", err)
		}
		return nil
	}
	return mount.Mount(mnt.Source, mnt.Destination, mnt.Type, mnt.Options)
}

func doAddDevice(pipe *os.File) error {
	msg := types.AddDeviceMsg{}
	if err := json.NewDecoder(pipe).Decode(&msg); err != nil {
		return err
	}

	force := msg.Force
	device := msg.Device

	existDev, err := DeviceFromPath(device.Path, "")
	// if device exists and device is the one we wantted, just return.
	if err == nil && existDev.Major == device.Major && existDev.Minor == device.Minor && existDev.Type == device.Type {
		// change filemode, uid and gid
		if err := os.Chmod(device.Path, device.FileMode); err != nil {
			logrus.Errorf("os.Chmod error: %v", err)
		}
		if err := os.Chown(device.Path, int(device.UID), int(device.GID)); err != nil {
			logrus.Errorf("os.Chown error: %v", err)
		}
		return nil
	}

	if force {
		// if force, the target file is not the one we wantted, just remove it.
		fmt.Printf("path %s in container already exists. removing it.\n", device.Path)
		if err := os.Remove(device.Path); err != nil {
			logrus.Errorf("os.Remove error: %v", err)
		}
	}

	// change umask
	oldMask := syscall.Umask(0000)
	defer syscall.Umask(oldMask)

	var needPermission []string
	dir := filepath.Dir(device.Path)
	for {
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			needPermission = append(needPermission, dir)
			dir = filepath.Dir(dir)
		} else {
			break
		}
	}

	dir = filepath.Dir(device.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	for _, dirname := range needPermission {
		if err := os.Chown(dirname, int(device.UID), int(device.GID)); err != nil {
			logrus.Errorf("os.Chown error: %v", err)
		}
	}

	if err := MknodDevice(device.Path, device); err != nil {
		return fmt.Errorf("Current OS kernel do not support mknod in container user namespace for root, err: %s", err)
	}
	return nil
}

func doRemoveDevice(pipe *os.File) error {
	var device types.Device
	if err := json.NewDecoder(pipe).Decode(&device); err != nil {
		return err
	}

	// As add-device supports `update-config-only` flag, it will update the config only.
	// So the device we wantted to remove maybe not exist in container at all, that's fine, just return OK.
	if _, err := os.Stat(device.Path); os.IsNotExist(err) {
		return nil
	}
	// if not a device.
	if _, err := DeviceFromPath(device.Path, ""); err != nil {
		return err
	}

	// need strict check here?
	return os.Remove(device.Path)
}

func doAddBind(pipe *os.File) error {
	var bind types.Bind
	if err := json.NewDecoder(pipe).Decode(&bind); err != nil {
		return err
	}

	var needPermission []string
	dir := filepath.Dir(bind.ContainerPath)
	for {
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			needPermission = append(needPermission, dir)
			dir = filepath.Dir(dir)
		} else {
			break
		}
	}

	if bind.IsDir {
		if err := os.MkdirAll(bind.ContainerPath, 0600); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(bind.ContainerPath), 0600); err != nil {
			return err
		}
		f, err := os.OpenFile(bind.ContainerPath, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("fail to create transfer path,err: %s", err)
		}
		f.Close()
	}

	if err := os.Chown(bind.ContainerPath, bind.UID, bind.GID); err != nil {
		logrus.Errorf("os.Chown error: %s", err)
	}
	for _, dirname := range needPermission {
		if err := os.Chown(dirname, bind.UID, bind.GID); err != nil {
			logrus.Errorf("os.Chown error: %s", err)
		}
	}

	if err := mount.Mount(bind.ResolvPath, bind.ContainerPath, "none", bind.MountOption); err != nil {
		return fmt.Errorf("fail to mount via transfer path: %+v, err: %s", bind, err)
	}
	return nil
}

func doRemoveBind(pipe *os.File) error {
	var bind types.Bind
	if err := json.NewDecoder(pipe).Decode(&bind); err != nil {
		return err
	}
	return mount.Unmount(bind.ContainerPath)
}

func doAddTransferBase(pipe *os.File) error {
	var bind types.Bind
	if err := json.NewDecoder(pipe).Decode(&bind); err != nil {
		return err
	}
	if err := os.MkdirAll(bind.ContainerPath, 0600); err != nil {
		return fmt.Errorf("doAddTransferBase: create transfer dir in container failed, err: %s", err)
	}
	if err := mount.Mount(bind.HostPath, bind.ContainerPath, "none", "ro,bind,rslave"); err != nil {
		return fmt.Errorf("doAddTransferBase: mount transfer dir in container failed, err:%s, %+v", err, bind)
	}

	return nil
}
