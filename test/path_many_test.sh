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

TEST_NAME="test_path_many"

test_001(){
	container_ID=`isula run --name one --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $container_ID
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "01:FAIL"
	fi
	
	container_ID=`isula ps | grep one | awk '{print $1}'`

	TEST_FOLDER1=$TMP/$TEST_NAME/001/test1
	TEST_FOLDER2=$TMP/$TEST_NAME/001/test2
	
	mkdir -p $TEST_FOLDER1
	mkdir -p $TEST_FOLDER2

	echo hello > $TEST_FOLDER1/b.txt
	echo cc > $TEST_FOLDER2/c.txt 
	
	$ISULAD_TOOLS add-path $container_ID $TEST_FOLDER2:/tmp:rw $TEST_FOLDER1:/home:rw > /dev/null
	out=`echo $?`
	if [ $out -ne 0 ]; then
		fail $TEST_NAME "01-1:FAIL"
	else
		success $TEST_NAME "01-1:PASS"
	fi
	out=`isula exec one bash -c "cat /tmp/c.txt"`
	if [ "$out" == "cc" ]; then
		success $TEST_NAME "01-2:PASS"
	else
		fail $TEST_NAME "01-2:FAIL"
	fi
	
	out=`isula exec one bash -c "cat /home/b.txt"`
	if [ "$out" == "hello" ]; then
		success $TEST_NAME "01-3:PASS"
	else
		fail $TEST_NAME "01-3:FAIL"
	fi

	#test remove-path
	$ISULAD_TOOLS remove-path $container_ID $TEST_FOLDER2:/tmp:rw $TEST_FOLDER1:/home:rw > /dev/null
	out=`isula exec one bash -c "ls /tmp && ls /home"`
	if [ "$out" == "" ]; then
		success $TEST_NAME "01-4:PASS"
	else
		fail $TEST_NAME "01-4:FAIL"
	fi

	#test mount different dirct to  container of the same direct
	$ISULAD_TOOLS add-path $container_ID  $TEST_FOLDER2:/tmp:rw $TEST_FOLDER1:/tmp:rw > /dev/null
	if [ $? -ne 0 ]; then
		fail $TEST_NAME "01-5:FAIL"
	else
		success $TEST_NAME "01-6:PASS"
	fi
	out=`isula exec $container_ID bash -c "ls /tmp"`
	if [ "$out" == "b.txt" ]; then
		success $TEST_NAME "01-7:PASS"
	else
		fail $TEST_NAME "01-7:FAIL"
	fi
	
	isula rm -f one > /dev/null
}

test_002(){
	#test Multiple container mount the same direct  
	container_ID1=`isula run --name one1 --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 1000"`
	container_status $container_ID1
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "02:FAIL"
	fi
	container_ID2=`isula run --name two --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 1000"`
	container_status $container_ID2
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "021:FAIL"
	fi
	TEST_FOLDER1=$TMP/$TEST_NAME/002
	mkdir -p $TEST_FOLDER1
	echo hello > $TEST_FOLDER1/b.txt

	$ISULAD_TOOLS add-path $container_ID1 $TEST_FOLDER1:/tmp:rw > /dev/null
	out=`isula exec one1 sh -c "ls /tmp"`
	if [ "$out" != "b.txt" ]; then
		fail $TEST_NAME "02-1:FAIL"
	else
		success $TEST_NAME "02-1:PASS"
	fi

	$ISULAD_TOOLS add-path $container_ID2 $TEST_FOLDER1:/tmp:rw > /dev/null
	out1=`isula exec two sh -c "cd tmp && ls"`
	if [ "$out1" != "b.txt" ]; then
		fail $TEST_NAME "02-2:FAIL"
	else
		success $TEST_NAME "02-2:PASS"
	fi 
	$ISULAD_TOOLS remove-path $container_ID1 $TEST_FOLDER1:/tmp:ro > /dev/null
	out=`echo $?`
	if [ $out -ne 0 ];then
		fail $TEST_NAME "02-3:FAIL"
	else
		success $TEST_NAME "02-3:PASS"
	fi

	out1=`isula exec two sh -c "cd tmp && ls"`
	if [ "$out1" == "b.txt" ]; then
		success $TEST_NAME "02-4:PASS"
	else
		fail $TEST_NAME "02-4:FAIL"
	fi

	#test stop start 
	isula stop two > /dev/null
	container_status $container_ID2
	if [ "${status}x" != "exitedx" ]; then
		fail $TEST_NAME "02-5:FAIL"
	fi
	isula start two > /dev/null
	container_status $container_ID2
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "02-6:FAIL"
	fi
	out1=`isula exec two sh -c "cd tmp && ls"`
	if [ "$out1" == "b.txt" ]; then
		success $TEST_NAME "02-7:PASS"
	else
		fail $TEST_NAME "02-7:FAIL"
	fi 
	isula rm -f one1 > /dev/null
	isula rm -f two > /dev/null
}

test_003(){
	#test one direct is ro ,the other is direct is rw
	out=`isula run --name one --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 1000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "03:FAIL"
	fi

	TEST_FOLDER1=$TMP/$TEST_NAME/003/test1
	TEST_FOLDER2=$TMP/$TEST_NAME/003/test2
	
	mkdir -p $TEST_FOLDER1
	mkdir -p $TEST_FOLDER2

	echo hello > $TEST_FOLDER1/b.txt
	echo cc > $TEST_FOLDER2/c.txt 

	$ISULAD_TOOLS add-path $out $TEST_FOLDER1:/tmp:rw $TEST_FOLDER2:/home:ro > /dev/null 2>&1
	out1=`isula exec one bash -c "cat /tmp/b.txt"`
	if [ "$out1" != "hello" ]; then
		fail $TEST_NAME "03-1:FAIL"
	fi 
	out1=`isula exec one bash -c "cat /home/c.txt"`
	if [ "$out1" != "cc" ]; then
		fail $TEST_NAME "03-2:FAIL"
	fi
	
	isula exec one bash -c "cd /home && echo hello>c.txt" > /dev/null 2>&1
	if [ $? -eq 0 ]; then
		fail $TEST_NAME "03-3:FAIL"
	fi
	out=`isula exec one bash -c "cd /home && cat c.txt"`
	if [ "$out" == "cc" ]; then
		success $TEST_NAME "03-4:PASS"
	else
		fail $TEST_NAME "03-4:FAIL"
	fi
	isula rm -f one > /dev/null
}

main(){
	test_001
	test_002
	test_003
}
main
