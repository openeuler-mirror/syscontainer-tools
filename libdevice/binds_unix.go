// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: bind operation for device
// Author: zhangwei
// Create: 2018-01-18

package libdevice

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/opencontainers/runtime-spec/specs-go"
	"isula.org/syscontainer-tools/types"
	"isula.org/syscontainer-tools/utils"
)

// ParseBind will parse host path to Bind structure
func ParseBind(bindstr string, spec *specs.Spec, create bool) (*types.Bind, error) {
	var src, dst string
	var isDir bool

	permissions := "rw,rslave"
	arr := strings.Split(bindstr, ":")
	switch len(arr) {
	case 3:
		if validMountOption(arr[2]) == false {
			return nil, fmt.Errorf("invalid permissions: %s", arr[2])
		}
		permissions = arr[2]
		fallthrough
	case 2:
		src = path.Clean(arr[0])
		dst = path.Clean(arr[1])
	default:
		return nil, fmt.Errorf("invalid path specification: %s", bindstr)
	}

	if path.IsAbs(src) == false || path.IsAbs(dst) == false {
		return nil, fmt.Errorf("invalid path specification:%s, only absolute path is allowed", bindstr)
	}

	bind := &types.Bind{
		HostPath:      src,
		ContainerPath: dst,
		MountOption:   permissions,
	}

	symlinkinfo, err := os.Lstat(src)
	if err != nil {
		logrus.Errorf("Lstat returns a FileInfo describing the named file error: %v", err)
	}
	info, err := os.Stat(src)
	if symlinkinfo != nil {
		if ((symlinkinfo.Mode() & os.ModeSymlink) == os.ModeSymlink) && (err != nil) {
			return nil, fmt.Errorf("Parsebind get symlibk source file error: %v", err)
		}
	}

	if create {
		if spec != nil {
			uid, gid := utils.GetUIDGid(spec)
			if uid == -1 {
				uid = 0
			}
			if gid == -1 {
				gid = 0
			}
			bind.UID = uid
			bind.GID = gid
		}
		if err == nil {
			isDir = info.IsDir()
		} else if os.IsNotExist(err) {
			isDir = true
			if spec != nil {
				if err := os.MkdirAll(src, os.FileMode(0755)); err != nil {
					return nil, fmt.Errorf("ParseBind mkdir error: %v", err)
				}
				if err := os.Chown(src, bind.UID, bind.GID); err != nil {
					return nil, fmt.Errorf("ParseBind chown error: %v", err)
				}
			}
		} else {
			return nil, fmt.Errorf("invalid path specification: %s", bindstr)
		}
	} else {
		if err == nil {
			isDir = info.IsDir()
		} else if os.IsNotExist(err) {
			return bind, nil
		} else {
			return nil, fmt.Errorf("invalid path specification: %s", bindstr)
		}
	}
	bind.IsDir = isDir

	return bind, nil
}

func findPathDevice(path string) (*types.Device, string, error) {

	// find path mount entry point.
	cmd := exec.Command("df", "-P", path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}

	defer cmd.Wait()

	reader := bufio.NewReader(stdout)

	// ignore first line.
	reader.ReadString('\n')
	line, err := reader.ReadString('\n')
	if err != nil {
		logrus.Errorf("reader.ReadString error: %v", err)
	}
	line = strings.Trim(line, "\n")

	devs := strings.Split(line, " ")
	device, err := DeviceFromPath(devs[0], "rwm")
	if err != nil {
		return nil, "", err
	}
	return device, devs[len(devs)-1], nil
}

func findDeviceMountEntryPoint(device, mp string) (string, string, string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", "", "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	procMntCols := 6
	for scanner.Scan() {
		line := scanner.Text()
		array := strings.Split(line, " ")
		if len(array) < procMntCols {
			continue
		}
		// the one we wanted.
		if array[0] == device && array[1] == mp {
			entry := array[1]
			fstype := array[2]
			mountOption := array[3]
			return entry, fstype, mountOption, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", "", err
	}
	return "", "", "", fmt.Errorf("Device Not Found")
}

// validMountOption will validate the mount option for user input
func validMountOption(option string) bool {
	validOp := map[string]bool{
		"ro": true,
		"rw": true,
		// "shared": true,
		"private": true,
		// "slave": true,
		// "rshared": true,
		"rprivate": true,
		"rslave":   true,
	}

	arr := strings.Split(option, ",")
	for _, op := range arr {
		if !validOp[op] {
			return false
		}
		validOp[op] = false
	}
	return true
}
