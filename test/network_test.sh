# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# syscontainer-tools is licensed under the Mulan PSL v2.
# You can use this software according to the terms and conditions of the Mulan PSL v2.
# You may obtain a copy of Mulan PSL v2 at:
#    http://license.coscl.org.cn/MulanPSL2
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v2 for more details.
# Description: network test
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash
############################################################################
#
#This script is to test
#
############################################################################

. $CUR/env.sh
. $CUR/tools.sh

test_001(){
	#testcase01
	CONTAINER_ID=`isula run -d $BUSYBOX_IMAGE top`
	$ISULAD_TOOLS --debug --log $TMP/syscontainer-tools.log add-nic \
		--type veth --name eth10 --ip 192.168.182.2/24 \
		--mac "aa:bb:cc:dd:ee:aa" --bridge "docker0" --mtu 1350 \
		$CONTAINER_ID
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-1:FAIL"
	else
		success $TEST_NAME "01-1:PASS"
	fi

	out=`isula exec $CONTAINER_ID ip a s eth10`
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-2:FAIL"
	else
		success $TEST_NAME "01-2:PASS"
	fi

	echo $out | grep "192.168.182.2/24" > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-3:FAIL"
	else
		success $TEST_NAME "01-3:PASS"
	fi

	echo $out | grep "aa:bb:cc:dd:ee:aa" > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-4:FAIL"
	else
		success $TEST_NAME "01-4:PASS"
	fi

	echo $out | grep "1350" > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-5:FAIL"
	else
		success $TEST_NAME "01-5:PASS"
	fi

	# check if the bridge contains veth nic
	brctl show docker0 | grep -E "veth[a-z0-9]{10}" > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-6:FAIL"
	else
		success $TEST_NAME "01-6:PASS"
	fi
	isula rm -f $CONTAINER_ID > /dev/null 2>&1
}

test_002(){
	# testcase02
	# test ovs bridge
	OVS_BR=test_ovs_bridge
	ovs-vsctl --if-exists del-br $OVS_BR
	ovs-vsctl add-br $OVS_BR
	ovs-vsctl br-exists $OVS_BR
	if [ $? -ne 0 ]; then
		fail "02-1:FAIL"
	fi
	CONTAINER_ID=`isula run -d $BUSYBOX_IMAGE top`
	$ISULAD_TOOLS --debug --log $TMP/syscontainer-tools.log add-nic \
		--type veth --name eth11 --ip 192.168.182.2/24 \
		--mac "aa:bb:cc:dd:ee:aa" --bridge $OVS_BR --mtu 1350 \
		$CONTAINER_ID
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "02-1:FAIL"
	else
		success $TEST_NAME "02-1:PASS"
	fi

	out=`isula exec $CONTAINER_ID ip a s eth11`
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "02-2:FAIL"
	else
		success $TEST_NAME "02-2:PASS"
	fi


	# check if the bridge contains veth nic
	ovs-vsctl list-ports $OVS_BR | grep -E "veth[a-z0-9]{10}" > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "02-3:FAIL"
	else
		success $TEST_NAME "02-3:PASS"
	fi
	isula rm -f $CONTAINER_ID > /dev/null 2>&1
	ovs-vsctl --if-exists del-br $OVS_BR
}

main(){
	test_001
	test_002
}

main

