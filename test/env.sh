# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: env tests
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash

## isulad-tools paths.
ISULAD_TOOLS="$CUR/../build/isulad-tools"

## Ubuntu image
UBUNTU_IMAGE="ubuntu"

## busybox image:
BUSYBOX_IMAGE="busybox"

## tmp directory:
TMP=$CUR/tmpdir

## block device:
DEV_SDA=/dev/sda
DEV_SDA1=/dev/sda
DEV_SDA2=/dev/zero
DEV_NOT_EXIST=/dev/not_exist_at_all
