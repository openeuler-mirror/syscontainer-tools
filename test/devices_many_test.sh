# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: test mount mutiple direct
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash
#test mount mutiple direct

. $CUR/env.sh
. $CUR/tools.sh

TEST_NAME="test_devices_many"
test_001(){
	container_ID=`lcrc run --name one  --hook-spec /var/lib/lcrd/hooks/hookspec.json  -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $container_ID
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "01:FAIL"
	fi
	container_ID=`lcrc ps | grep one | awk '{print $1}'`
	$ISULAD_TOOLS add-device $container_ID $DEV_SDA1:/dev/sda1:rw $DEV_SDA2:/dev/sda2:rw > /dev/null
	out=`lcrc exec one bash -c "ls /dev/sda1"`
	if [ "$out" == "/dev/sda1" ]; then
		success $TEST_NAME "01-1:PASS"
	else
		fail $TEST_NAME "01-1:FAIL"
	fi
	
	out=`lcrc exec one bash -c "ls /dev/sda2"`
	if [ "$out" == "/dev/sda2" ]; then
		success $TEST_NAME "01-2:PASS"
	else
		fail $TEST_NAME "01-2:FAIL"
	fi
	#test remove-device
	$ISULAD_TOOLS remove-device $container_ID  $DEV_SDA1:/dev/sda1:rw /dev/zero:/dev/sda2:rw > /dev/null
	lcrc exec one bash -c "ls /dev/sda1 && /dev/sda2" >&$TEST_FOLDER/ab.txt
	out=`cat $TEST_FOLDER/ab.txt`
	out=${out##*:}
	out=${out%%or*}
	if [ "$out" == " No such file " ]; then
		success $TEST_NAME "01-3:PASS"
	else
		fail $TEST_NAME "01-3:FAIL"
	fi

	#test mount different dirct to  container of the same direct
	$ISULAD_TOOLS add-device $container_ID  $DEV_SDA1:/dev/sda1:rw /dev/zero:/dev/sda1:rw>& $TEST_FOLDER/ab.txt
	out=`cat $TEST_FOLDER/ab.txt |  awk -F: 'END{print $1}'`
	if [ "$out" == "Failed to add device" ]; then
		success $TEST_NAME "01-4:PASS"
	else
		fail $TEST_NAME "01-4:FAIL"
	fi
	lcrc rm -f one > /dev/null
}

test_002(){
	#test Multiple container mount the same direct  
	container_ID1=`lcrc run --name one --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE bash  -c "sleep 10000"`
	container_status $container_ID1
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "02:FAIL"
	fi
	container_ID2=`lcrc run --name two --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $container_ID2
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "021:FAIL"
	fi
	
	out=`$ISULAD_TOOLS add-device $container_ID1  $DEV_SDA1:/dev/sda1:rw > /dev/null`
	out=`lcrc exec  one bash -c "ls /dev/sda1"`
	if [ "$out" != "/dev/sda1" ]; then
		fail $TEST_NAME "02-1:FAIL"
	else
		success $TEST_NAME "02-1:PASS"
	fi 
	
	out=`$ISULAD_TOOLS add-device $container_ID2  $DEV_SDA1:/dev/sda1:rw > /dev/null`
	out1=`lcrc exec two bash -c "ls /dev/sda1"`
	if [ "$out1" != "/dev/sda1" ]; then
		fail $TEST_NAME "02-2:FAIL"
	else
		success $TEST_NAME "02-2:PASS"
	fi 
	
	$ISULAD_TOOLS remove-device $container_ID1 $DEV_SDA1:/dev/sda1:rw > /dev/null
	lcrc exec $container_ID1 bash -c "ls /dev/sda1" > /dev/null 2>&1
	out=`echo $?`
	if [ $out -eq 0 ];then
		fail $TEST_NAME "02-3:FAIL"
	else
		success $TEST_NAME "02-3:PASS"
	fi
	
	out1=`lcrc exec two bash -c "ls /dev/sda1"`
	if [ "$out1" == "/dev/sda1" ]; then
		success $TEST_NAME "02-4:PASS"
	else
		fail $TEST_NAME "02-4:FAIL"
	fi

	#test stop start 
	lcrc stop two > /dev/null
	container_status $container_ID2
	if [ "${status}x" != "exitedx" ]; then
		fail $TEST_NAME "02-5:FAIL"
	fi
	lcrc start two > /dev/null
	container_status $container_ID2
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "02-6:FAIL"
	fi
	out1=`lcrc exec two bash -c "ls /dev/sda1"`
	if [ "$out1" == "/dev/sda1" ]; then
		success $TEST_NAME "02-7:PASS"
	else
		fail $TEST_NAME "02-7:FAIL"
	fi 
	lcrc rm -f one > /dev/null
	lcrc rm -f two > /dev/null
}



main(){
	test_001
	test_002
}

main
