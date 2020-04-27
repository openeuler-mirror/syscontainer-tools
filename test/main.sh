# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# syscontainer-tools is licensed under the Mulan PSL v2.
# You can use this software according to the terms and conditions of the Mulan PSL v2.
# You may obtain a copy of Mulan PSL v2 at:
#    http://license.coscl.org.cn/MulanPSL2
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v2 for more details.
# Description: main test
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash


## current directory:
export CUR=$(cd `dirname $0`; pwd)
. $CUR/env.sh

declare -i total_cases=0
declare -i failed_cases=0
declare -i success_cases=0
export total_cases
export failed_cases
export success_cases

setupImage(){
	declare -a Images=($UBUNTU_IMAGE $BUSYBOX_IMAGE)
	for img in "${Images[@]}";
	do
		out=`isula images | grep $img`
		if [ "x$out" = "x" ]; then
			echo "Image [" $img "] does not exist, pull it from hub."
			isula pull $img
		fi
	done

}


setup_device_hook(){
	mkdir -p /var/lib/isulad/hooks
	cp $CUR/../hooks/syscontainer-hooks/example/hookspec.json /var/lib/isulad/hooks/
	cp $CUR/../build/syscontainer-hooks /var/lib/isulad/hooks/
}

main_test(){
	. $CUR/devices_test.sh
	. $CUR/devices_many_test.sh
	. $CUR/path_test.sh
	. $CUR/path_many_test.sh
	. $CUR/network_test.sh
	. $CUR/route_test.sh
}

report(){
	echo "============ Result =========="
	echo "total cases  :" $total_cases
	echo "failed cases :" $failed_cases
	echo "success cases:" $success_cases
}

main(){
	mkdir -p $TMP
	setupImage
	setup_device_hook
	main_test
	rm -rf $TMP

	# report the result
	report
	exit $failed_cases
}

main
