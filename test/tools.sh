# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# syscontainer-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: test tools
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash

container_status(){
        id=$1
        status=`isula inspect ${id:00:12} | grep Status | awk -F ":" '{print $2}'`
        status=${status#*\"}
        status=${status%%\"*}
}

fail(){
	total_cases=$((total_cases+1))
	failed_cases=$((failed_cases+1))
	echo $@
}

success(){
	total_cases=$((total_cases+1))
	success_cases=$((success_cases+1))
	echo $@
}
