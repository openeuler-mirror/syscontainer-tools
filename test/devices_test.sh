# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: device tests
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash

. $CUR/env.sh
. $CUR/tools.sh

TEST_NAME="test_devices"

test_001(){
	# test add-device.
	out=`lcrc run --name one --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 100000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "01:FAIL"
	fi
	
	$ISULAD_TOOLS add-device $out $DEV_SDA1:/dev/sda1:rw> /dev/null
	out1=`lcrc exec one sh -c "ls /dev/sda1"`
	if [ "$out1" == "/dev/sda1" ]; then
		success $TEST_NAME "01-1:PASS"
	else
		fail $TEST_NAME "01-1:FAIL"
	fi

	#test remove-device
	$ISULAD_TOOLS remove-device $out $DEV_SDA1:/dev/sda1:rwm > /dev/null
	out1=`lcrc exec one sh -c "ls -l /dev/sda1" > /dev/null 2>&1`
	if [ "$out1" == "" ]; then
		success $TEST_NAME "01-2:PASS"
	else
		fail $TEST_NAME "01-2:FAIL"
	fi
	
	lcrc rm -f one > /dev/null
}

test_002(){
	#test exited container
	out=`lcrc run --name one --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE`
	sleep 3
	container_status $out
	if [ "${status}x" != "exitedx" ]; then
		fail $TEST_NAME "02:FAIL" $out $status
	fi
	$ISULAD_TOOLS add-device $out $DEV_SDA1:/dev/sda1:rwm >&`pwd`/ab.txt
	out1=`cat ab.txt | awk -F: 'END{print $1}'`
	if [ "$out1" == "Failed to add device" ];  then
		success $TEST_NAME "02-1:PASS"
	else
		fail $TEST_NAME "02-1:FAIL"
	fi
	lcrc rm one > /dev/null
	rm -f ab.txt >/dev/null
}

test_003(){
	#test created container
	out=`lcrc create --name one --hook-spec /var/lib/lcrd/hooks/hookspec.json -ti  $UBUNTU_IMAGE`
	container_status $out
	if [ "${status}x" != "createdx" ]; then
		fail $TEST_NAME "03:FAIL"
	fi
	$ISULAD_TOOLS add-device $out $DEV_SDA1:/dev/sda1:rwm >&`pwd`/ab.txt
	out1=`cat ab.txt | awk -F: 'END{print $1}'`
	if [ "$out1" == "Failed to add device" ]; then
		success $TEST_NAME "03-1:PASS"
	else
		fail $TEST_NAME "03-1:FAIL"
	fi
	rm -f ab.txt > /dev/null
	
	#created->up container
	lcrc start one > /dev/null
	out=`lcrc ps | grep one | awk '{print $1}'`
	out1=`$ISULAD_TOOLS add-device $out $DEV_SDA1:/dev/sda1:rwm`
	out1=`lcrc exec one sh -c "ls /dev/sda1"`
	
	if [ "$out1" == "/dev/sda1" ]; then
		success $TEST_NAME "03-2:PASS"
	else
		fail $TEST_NAME "03-2:FAIL"
	fi
	lcrc rm -f one > /dev/null
}

test_004(){
	#test r
	out=`lcrc run --name one --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "04:FAIL"
	fi
	
	$ISULAD_TOOLS add-device $out $DEV_SDA1:/dev/sda1:r > /dev/null
	out1=`lcrc exec one sh -c "ls  /dev/sda1"`
	if [ "$out1" != "/dev/sda1" ]; then
		fail $TEST_NAME "04-1:FAIL"
	else
		success $TEST_NAME "04-1:PASS"
	fi
	$ISULAD_TOOLS remove-device $out $DEV_SDA1:/dev/sda1:r > /dev/null
	out=`lcrc exec one sh -c "ls  /dev/sda1" > /dev/null 2>&1`
	if [ "$out" == "" ]; then
		success $TEST_NAME "04-2:PASS"
	else
		fail $TEST_NAME "04-2:FAIL"
	fi
	rm -rf ab.txt > /dev/null

	#test rw
	out=`lcrc ps | grep one | awk '{print $1}'`
	$ISULAD_TOOLS add-device $out $DEV_SDA1:/dev/sda1:rw > /dev/null
	out=`lcrc exec one sh -c "ls  /dev/sda1"`
	if [ "$out" != "/dev/sda1" ]; then
		fail $TEST_NAME "04-3:FAIL"
	else
		success $TEST_NAME "04-3:PASS"
	fi
	lcrc exec one bash  -c "dd if=/dev/sda1 of=/dev/null bs=1M count=10" >&`pwd`/ab.txt
	out=`cat ab.txt | awk -F',' 'END{print $1}'`
	out=`echo $out | awk -F ' ' '{print $1}'`
	if [ "$out" == "10485760" ]; then
		success $TEST_NAME "04-4:PASS"
	else
		fail $TEST_NAME "04-4:FAIL"
	fi
	lcrc rm -f one > /dev/null
	rm -f ab.txt > /dev/null
}

test_006(){
	#test not exist device
	out=`lcrc run --name one  --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 100000"`
	container_status $out
	if [ "${status}x" == "runningx" ]; then
		success $TEST_NAME "06:PASS"
	else
		fail $TEST_NAME "06:FAIL"
	fi
	$ISULAD_TOOLS add-device $out $DEV_NOT_EXIST:/dev/sda1:rw >&`pwd`/ab.txt
	out=`cat ab.txt | awk -F: 'END{print $1}'`
	if [ "$out" == "Failed to parse device" ]; then
		success $TEST_NAME "06-1:PASS"
	else
		fail $TEST_NAME "06-1:FAIL"
	fi
	rm -f ab.txt > /dev/null
	lcrc rm -f one > /dev/null

	#test no r w
	out=`lcrc run --name one --hook-spec /var/lib/lcrd/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 100000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "06-2:FAIL"
	else
		success $TEST_NAME "06-2:PASS"
	fi
	$ISULAD_TOOLS add-device $out $DEV_SDA:/dev/sda > /dev/null
	lcrc exec one bash  -c "dd if=/dev/sda of=/dev/null bs=1M count=10" >&$TMP/ab.txt
	out=`cat $TMP/ab.txt | awk -F',' 'END{print $1}'`
	out=`echo $out | awk -F ' ' '{print $1}'`
	if [ "$out" == "10485760" ]; then
		success $TEST_NAME "06-3:PASS"
	else
		fail $TEST_NAME "06-3:FAIL"
	fi
	lcrc rm -f one > /dev/null
}


main(){
	test_001
	test_002
	test_003
	test_004
	test_006
}

main
