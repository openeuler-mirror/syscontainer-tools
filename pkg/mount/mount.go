// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//    http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Description: recursive unmount
// Author: zhangwei
// Create: 2018-01-18

package mount

import (
	"sort"
	"strings"

	docker_mount "github.com/docker/docker/pkg/mount"
)

// RecursiveUnmount unmounts the target and all mounts underneath, starting with
// the deepsest mount first.
func RecursiveUnmount(target string) error {
	mounts, err := docker_mount.GetMounts()
	if err != nil {
		return err
	}

	// Make the deepest mount be first
	sort.Sort(sort.Reverse(byMountpoint(mounts)))

	for i, m := range mounts {
		if !strings.HasPrefix(m.Mountpoint, target) {
			continue
		}
		if err := docker_mount.Unmount(m.Mountpoint); err != nil && i == len(mounts)-1 {
			if mounted, err := docker_mount.Mounted(m.Mountpoint); err != nil || mounted {
				return err
			}
			// Ignore errors for submounts and continue trying to unmount others
			// The final unmount should fail if there ane any submounts remaining
		}
	}
	return nil
}
