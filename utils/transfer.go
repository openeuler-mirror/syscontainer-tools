// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: mount transfer utils
// Author: zhangwei
// Create: 2018-01-18

package utils

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	mymount "isula.org/isulad-tools/pkg/mount"
	"github.com/docker/docker/pkg/mount"

	"isula.org/isulad-tools/types"
	"github.com/sirupsen/logrus"
)

const (
	masterPath      = "/.sharedpath/master"
	midTransferPath = "/.sharedpath/midpath"
	slavePath       = "/.sharedpath"
)

/* Add path to container when it is running

   We use mount propagation mechanism to do it. Take
   isulad tools add-path /hostpath1:/guest1 for example.


   1. Add a sharing path using hook as belowing. Then every
      new mount event will propagate to container.

	container ---->/.sharedpath (rslave,ro)
	host       --->/.sharedpath/master/containerid (rshared,rw)

   2. Add transfer path

      a.  (host)mount  --bind -o rw  /host1 /.sharedpath/midpath/containerid/hostpath1
      b.  (host)mount  --bind -o rw  /.sharedpath/midpath/containerid/hostpath1 /.sharedpath/master/containerid/hostpath1
      c.  (container) mount --bind -ro /.sharedpath/hostpath1 /guest1
*/

func releaseMountpoint(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if err := mymount.Unmount(path); err != nil {
		logrus.Errorf("releaseMountpoint: Failed to umount: %s, error: %s, still try to remove path", path, err)
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return nil

}

// RemoveTransferPath remove transfer path
func RemoveTransferPath(id string, bind *types.Bind) error {
	midPath, tarsferPath := getTransferPath(id, bind.HostPath)
	if err := releaseMountpoint(midPath); err != nil {
		logrus.Errorf("RemoveTransferPath failed: %s", err)
	}
	if err := releaseMountpoint(tarsferPath); err != nil {
		logrus.Errorf("RemoveTransferPath failed: %s", err)
	}
	return nil
}

// RemoveContainerSpecPath remove container spec
func RemoveContainerSpecPath(id string) error {
	if err := os.RemoveAll(GetContainerMidDir(id)); err != nil {
		logrus.Errorf("RemoveSharedPath failed, err: %s", err)
	}

	if err := os.RemoveAll(GetContainerSpecDir(id)); err != nil {
		logrus.Errorf("RemoveSharedPath failed, err: %s", err)
	}
	return nil
}

func parepareMountpoint(sPath, dPath, mOpt string, isDir bool) error {
	if isDir {
		if err := os.MkdirAll(dPath, 0600); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(dPath), 0600); err != nil {
			return err
		}
		f, err := os.OpenFile(dPath, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("Fail to create transfer path,err: %s", err)
		}
		f.Close()
	}

	if m, err := mount.Mounted(dPath); err != nil {
		return fmt.Errorf("Failed to mount path %s, err: %s", dPath, err)
	} else if m == true {
		return nil
	}
	return mymount.Mount(sPath, dPath, "none", mOpt)
}

// PrepareTransferPath prepares the transfer path for sharing.
// To propagate mount options to container, we need two middle paths.
func PrepareTransferPath(containerPath, id string, bind *types.Bind, doMount bool) error {
	midpath, tarsferPath := getTransferPath(id, bind.HostPath)
	bind.MountOption += ",bind"
	bind.ResolvPath = filepath.Join(containerPath, getRelativePath(bind.HostPath))
	if !doMount {
		return nil
	}

	// 1. check mount propagation
	if err := mymount.ValidMountPropagation(bind.HostPath, bind.MountOption); err != nil {
		return err
	}

	// 2. prepare midpath
	if err := parepareMountpoint(bind.HostPath, midpath, bind.MountOption, bind.IsDir); err != nil {
		return err
	}

	// 3. prapare transferpath
	if err := parepareMountpoint(midpath, tarsferPath, bind.MountOption, bind.IsDir); err != nil {
		return err
	}
	return nil

}

func getRelativePath(hostpath string) string {
	return filepath.Join(slavePath, getTransferBase(hostpath))
}
func getTransferPath(id, hostpath string) (string, string) {
	transfer := filepath.Join(GetContainerSpecDir(id), getTransferBase(hostpath))
	midpath := filepath.Join(midTransferPath, id, getTransferBase(hostpath))
	return midpath, transfer
}
func getTransferBase(path string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(path)))
}

// GetContainerSpecDir get container spec dir
func GetContainerSpecDir(id string) string {
	return filepath.Join(masterPath, id)
}

// GetContainerMidDir get container middle dir
func GetContainerMidDir(id string) string {
	return filepath.Join(midTransferPath, id)
}

// GetSlavePath get slave path
func GetSlavePath() string {
	return slavePath
}

// PrepareHostPath prepare host path
func PrepareHostPath(id string) error {

	if err := os.MkdirAll(masterPath, 0600); err != nil {
		return fmt.Errorf("create host shared path failed, err: %s", err)
	}
	if m, _ := mount.Mounted(masterPath); m != true {
		if err := mount.Mount("none", masterPath, "tmpfs", "size=16m"); err != nil {
			return fmt.Errorf("mount host shared path failed:, %s", err)
		}
		if err := syscall.Mount("none", masterPath, "none", syscall.MS_SHARED|syscall.MS_REC, ""); err != nil {
			return fmt.Errorf("failed to make mountpoint shared, err: %s", err)
		}
	}

	if err := os.MkdirAll(filepath.Join(masterPath, id), 0600); err != nil {
		return fmt.Errorf("create host shared path failed, err: %s", err)
	}
	return nil
}
