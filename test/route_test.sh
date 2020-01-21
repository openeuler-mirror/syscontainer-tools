# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# syscontainer-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: test route
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash

. $CUR/env.sh
. $CUR/tools.sh

test_001(){
	BR=tool_br
	ip link delete $BR > /dev/null 2>&1
	brctl addbr $BR > /dev/null 2>&1
	ip link set $BR up
	ip a a 192.168.182.1/24 dev $BR
	
	CONTAINER_ID=`isula run -d --net none $BUSYBOX_IMAGE top`
	$ISULAD_TOOLS --debug --log $TMP/syscontainer-tools.log add-nic \
		--type veth --name eth0 --ip 192.168.182.2/24 \
		--mac "aa:bb:cc:dd:ee:aa" --bridge $BR --mtu 1450 \
		$CONTAINER_ID
	isula exec --privileged $CONTAINER_ID ip route delete 192.168.182.0/24
	$ISULAD_TOOLS add-route $CONTAINER_ID '[{"dest":"192.168.182.0/24", "src":"192.168.182.2","dev":"eth0"}]'
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-1:FAIL"
	else
		success $TEST_NAME "01-1:PASS"
	fi

	# TODO: default gw can't set, what's the problem?
	#$ISULAD_TOOLS add-route $CONTAINER_ID '[{"gw":"192.168.182.1","dev":"eth0"}]'
	#if [ $? -ne 0 ]; then
	#	fail $TEST_NAME "01-1:FAIL"
	#else
	#	success $TEST_NAME "01-1:PASS"
	#fi

	rules=`isula exec $CONTAINER_ID ip route`
	echo $rules | grep "192.168.182.0/24 dev eth0 src 192.168.182.2" > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-2:FAIL"
	else
		success $TEST_NAME "01-2:PASS"
	fi
	isula rm -f $CONTAINER_ID > /dev/null 2>&1
	brctl delbr $BR > /dev/null 2>&1
}

main(){
	test_001
}

main
