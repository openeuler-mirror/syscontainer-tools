// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: mount point compare implement
// Author: zhangwei
// Create: 2018-01-18

package mount

import docker_mount "github.com/docker/docker/pkg/mount"

type byMountpoint []*docker_mount.Info

func (by byMountpoint) Len() int {
	return len(by)
}

func (by byMountpoint) Less(i, j int) bool {
	return by[i].Mountpoint < by[j].Mountpoint
}

func (by byMountpoint) Swap(i, j int) {
	by[i], by[j] = by[j], by[i]
}
