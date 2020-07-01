// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: mount operation
// Author: zhangwei
// Create: 2018-01-18

package mount

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	docker_mount "github.com/docker/docker/pkg/mount"
)

// Mount is mount operation
func Mount(device, target, mType, options string) error {
	flagint, data := docker_mount.ParseOptions(options)
	flag := uintptr(flagint)
	// propagation option
	propagationFlags := (uintptr)(syscall.MS_SLAVE | syscall.MS_SHARED | syscall.MS_UNBINDABLE | syscall.MS_PRIVATE)

	if err := syscall.Mount(device, target, mType, flag&^propagationFlags, data); err != nil {
		return err
	}

	// If we have a bind mount or remount, remount...
	if flag&syscall.MS_BIND == syscall.MS_BIND && flag&syscall.MS_RDONLY == syscall.MS_RDONLY {
		return syscall.Mount(device, target, mType, flag|syscall.MS_REMOUNT, data)
	}
	if flag&propagationFlags != 0 {
		return syscall.Mount("none", target, "none", flag&propagationFlags, data)
	}
	return nil
}

// Unmount is unmount operation
func Unmount(target string) error {
	return syscall.Unmount(target, syscall.MNT_DETACH)
}

// ValidMountPropagation checks propagation of path
func ValidMountPropagation(path, mOpt string) error {
	var bind, slave, shared bool
	var slavemnt, sharedmnt bool
	for _, opt := range strings.Split(mOpt, ",") {
		if opt == "bind" {
			bind = true
		}
		if opt == "shared" {
			shared = true
		}
		if opt == "slave" {
			slave = true
		}
	}
	if !bind {
		return nil
	}

	source, options, err := getSource(path)
	if err != nil {
		return err
	}

	for _, opt := range strings.Split(options, " ") {
		if strings.HasPrefix(opt, "shared:") {
			sharedmnt = true
			break
		}
		if strings.HasPrefix(opt, "master:") {
			slavemnt = true
			break
		}
	}
	if shared && !sharedmnt {
		return fmt.Errorf("Path %s is mounted on %s but it is not a shared mount", path, source)
	}
	if slave && !sharedmnt && !slavemnt {
		return fmt.Errorf("Path %s is mounted on %s but it is not a shared or slave mount", path, source)
	}
	return nil
}

func getSource(sourcepath string) (string, string, error) {
	path, err := filepath.EvalSymlinks(sourcepath)
	if err != nil {
		return "", "", err
	}

	mountinfos, err := docker_mount.GetMounts()
	if err != nil {
		return "", "", err
	}

	for {
		for _, m := range mountinfos {
			if m.Mountpoint == path {
				return path, m.Optional, nil
			}
		}
		if path == "/" {
			return "", "", fmt.Errorf("Could not find mount %s", sourcepath)
		}
		path = filepath.Dir(path)
	}
	return "", "", fmt.Errorf("Unexpected error in getMouont")
}
