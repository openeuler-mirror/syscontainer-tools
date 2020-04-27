// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: selinux utils
// Author: zhangwei
// Create: 2018-01-18

package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	// InitPath is the path of container'init process
	InitPath = "/sbin/init"

	systemdInit      = "systemd"
	xattrNameSelinux = "security.selinux"
)

// SELinuxContext user:role:type:level
type SELinuxContext map[string]string

// IsExist judges whether a file exists
func IsExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// SeconfigGet gets the k/v that come from /etc/selinux/config,like 'SELINUX=permissive'
func SeconfigGet(path, key string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.HasPrefix(txt, "#") {
			continue
		}

		fields := strings.Split(txt, "=")
		if len(fields) == 2 && fields[0] == key {
			return fields[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("Parse SELinux config file err")
}

// SeconfigSet sets the k/v that come from /etc/selinux/config,like 'SELINUX=permissive'
func SeconfigSet(path, key, value string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bytes.NewBufferString("")
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.HasPrefix(txt, "#") {
			if _, err := buf.WriteString(txt + "\n"); err != nil {
				logrus.Errorf("buf.WriteString err: %v", err)
			}
			continue
		}

		fields := strings.Split(txt, "=")
		if len(fields) == 2 && fields[0] == key {
			if _, err := buf.WriteString(fmt.Sprintf("%s=%s\n", key, value)); err != nil {
				logrus.Errorf("buf.WriteString err: %v", err)
			}
			continue
		}
		if _, err := buf.WriteString(txt + "\n"); err != nil {
			logrus.Errorf("buf.WriteString err: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return ioutil.WriteFile(path, buf.Bytes(), 0600)
}

// BindMount creates bind mount
func BindMount(source, dest string, readonly bool) error {
	if err := syscall.Mount(source, dest, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}

	/*  Remount bind mount to read/only if requested by the caller */
	if readonly {
		if err := syscall.Mount(source, dest, "bind", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_REC, ""); err != nil {
			return err
		}
	}
	return nil
}

// GetSelinuxMountPount gets the path which selinuxfs will be mounted.
func GetSelinuxMountPount(rootfs string) string {
	selinuxMountPoint := []string{
		"/selinux",
		"/sys/fs/selinux",
	}
	for _, path := range selinuxMountPoint {
		if IsExist(rootfs + path) {
			return path
		}
	}
	return selinuxMountPoint[0]
}

// IsSystemdInit judges whether a systemd
func IsSystemdInit(rootfs string) bool {
	if initPath, err := filepath.EvalSymlinks(rootfs + InitPath); err == nil && strings.Contains(initPath, systemdInit) {
		return true
	}
	return false
}

// Fatal fatal err, need exit.
func Fatal(err error) {
	logrus.Error(err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// Get gets SELinuxContext string
func (c SELinuxContext) Get() string {
	return fmt.Sprintf("%s:%s:%s:%s", c["user"], c["role"], c["type"], c["level"])
}

// SetType sets SELinuxContext type
func (c SELinuxContext) SetType(t string) {
	c["type"] = t
}

// GetType gets SELinuxContext type
func (c SELinuxContext) GetType() string {
	return c["type"]
}

// NewContext creates a new SELinuxContext
func NewContext(scon string) SELinuxContext {
	c := make(SELinuxContext)
	// unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023
	if len(scon) != 0 {
		con := strings.SplitN(scon, ":", 4)
		c["user"] = con[0]
		c["role"] = con[1]
		c["type"] = con[2]
		c["level"] = con[3]
	}
	return c
}
