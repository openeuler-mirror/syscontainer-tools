# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: network wrapper
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash
ip addr add 10.0.0.100/24 dev mynet1
ip link set mynet1 up
ping -c 5 10.0.0.100
ping -c 5 10.0.0.1
