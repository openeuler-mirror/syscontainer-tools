# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: test up container
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash
#test up container

. $CUR/env.sh
. $CUR/tools.sh

TEST_NAME="test_path"

test_001(){
	out=`isula run --name one  --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 100000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		 fail $TEST_NAME "01:FAIL"
	fi

	out1=`isula ps | grep one | awk '{print $1}'`
	TEST_FOLDER=$TMP/$TEST_NAME/001
	if [ -d $TEST_FOLDER ]; then
		rm -rf $TEST_FOLDER > /dev/null
	fi
	mkdir -p $TEST_FOLDER

	echo hello > $TEST_FOLDER/b.txt
	
	$ISULAD_TOOLS add-path $out1 $TEST_FOLDER:/tmp:rw > /dev/null
	out=`echo $?`
	if [ $out -ne 0 ]; then
		fail $TEST_NAME "01-1:FAIL"
	fi
	
	out=`isula exec one sh -c "cat /tmp/b.txt"`
	if [ "$out" == "hello" ]; then
		success $TEST_NAME "01-2:PASS"
	else
		fail $TEST_NAME "01-2:FAIL"
	fi
	
	#test remove-path
	$ISULAD_TOOLS remove-path $out1 $TEST_FOLDER:/tmp > /dev/null
	out=`isula exec one sh -c "cd tmp && ls" > /dev/null 2>&1 `
	if [ "$out" == "" ]; then
		success $TEST_NAME "01-3:PASS"
	else
		fail $TEST_NAME "01-3:FAIL"
	fi

	# clean up container.
	isula rm -f one > /dev/null
}

test_002(){
	#test exited container
	out=`isula run --name one  --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE`
	sleep 3
	container_status $out
	if [ "${status}x" != "exitedx" ]; then
		fail $TEST_NAME "02:FAIL"
	fi

	$ISULAD_TOOLS add-path $out `pwd`:/tmp:rw > /dev/null 2>&1
	out=`echo $?`
	if [ $out -ne 0 ]; then
		success $TEST_NAME "02-1:PASS"
	else
		fail $TEST_NAME "02-1:Fail"
	fi
	isula rm one > /dev/null
}

test_003(){
	#test created container
	out=`isula create --name one  --hook-spec /var/lib/isulad/hooks/hookspec.json -ti  $UBUNTU_IMAGE`
	container_status $out
	if [ "${status}x" != "createdx" ]; then
		success $TEST_NAME "03:PASS"
	fi
	$ISULAD_TOOLS add-path $out `pwd`:/tmp:rw > /dev/null 2>&1
	out=`echo $?`
	if [ $out -ne 0 ]; then
		success $TEST_NAME "03-1:PASS"
	else
		fail $TEST_NAME "03-1:FAIL"
	fi

	TEST_FOLDER=$TMP/$TEST_NAME/003
	rm -rf $TEST_FOLDER > /dev/null
	mkdir -p $TEST_FOLDER

	echo hello > $TEST_FOLDER/b.txt

	#created->up container
	out1=`isula start one`
	sleep 1
	out=`isula ps | grep one | awk '{print $1}'`
	$ISULAD_TOOLS add-path $out $TEST_FOLDER:/tmp:ro > /dev/null 2>&1
	out1=`isula exec $out sh -c "cat /tmp/b.txt"`
	if [ "$out1" == "hello" ]; then
		success $TEST_NAME "03-2:PASS"
	else
		fail $TEST_NAME "03-2:FAIL"
	fi

	#test ro 
	isula exec one sh -c "cd tmp && ls && echo abcddd> b.txt" > /dev/null 2>&1 
	out=`echo $?`
	if [ $out -ne 0 ]; then
		success $TEST_NAME "03-2:PASS"
	else
		fail $TEST_NAME "03-2:FAIL"
	fi

	out=`isula ps | grep one | awk '{print $1}'`
	isula rm -f one > /dev/null
}

test_005(){
	#test mount a Empty dirct
	out=`isula run --name one --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "05:FAIL"
	fi
	mkdir -p /tmp/isulad_test/test
	$ISULAD_TOOLS add-path $out /tmp/isulad_test/test:/tmp:rw > /dev/null 2>&1
	out1=`isula exec one bash -c "mount | awk 'END{print $1}'"`
	out1=${out1%on*}
	out2=${out1##*/}
	if [ "$out1" == "/dev/$out2" ]; then
		success $TEST_NAME "05-1:PASS"
	else
		fail $TEST_NAME "05-1:FAIL"
	fi
	isula rm -f one > /dev/null
}

test_006(){
	#test can not add ro and rw
	out=`isula run --name one --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "06:FAIL"
	fi
	out=`isula exec $out sh -c "cd tmp && ls && echo cc > b.txt && cat b.txt"`
	if [ "$out" == "cc" ]; then
		success $TEST_NAME "06-1:PASS"
	else
		fail $TEST_NAME "06-1:FAIL"
	fi
	isula rm -f one > /dev/null
}

test_007(){
	#test remove dirct
	out=`isula run --name one --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 10000"`
	container_status $out
	if [ "${status}x" != "runningx" ]; then
		fail $TEST_NAME "07:FAIL"
	fi

	TEST_FOLDER=$TMP/$TEST_NAME/007
	if [ -d $TEST_FOLDER ]; then
		rm -rf $TEST_FOLDER > /dev/null
	fi
	mkdir -p $TEST_FOLDER

	echo hello > $TEST_FOLDER/b.txt

	$ISULAD_TOOLS add-path $id $TEST_FOLDER:/tmp:rw > /dev/null
	out=`echo $?`
	if [ $out -ne 0 ]; then
		fail $TEST_NAME "07-1:FAIL"
	fi
	out=`isula exec one bash -c "cat /tmp/b.txt"`
	if [ "$out" != "hello" ]; then
		fail $TEST_NAME "07-2:FAIL"
	fi
	
	# remove the path from container.
	$ISULAD_TOOLS remove-path one $TEST_FOLDER:/tmp:rw > /dev/null
	out=`isula exec one bash -c "ls -l /tmp"`
	if [ "$out" == "total 0" ]; then
		success $TEST_NAME "07-3:PASS"
	else
		fail $TEST_NAME "07-3:FAIL"
	fi

	# clean up container.
	isula rm -f one > /dev/null
}

test_008(){
	out=`isula run --name one  --hook-spec /var/lib/isulad/hooks/hookspec.json -d $UBUNTU_IMAGE bash -c "sleep 100000"`
	out2=`isula ps | grep one | awk '{print $1}'`
	$ISULAD_TOOLS add-path $out2 $out1:/tmp:rw > /dev/null 2>&1
	out=`echo $?`
	if [ $out -ne 0 ]; then
		success $TEST_NAME "08-1:PASS"
	else
		fail $TEST_NAME "08-1:FAIL"
	fi
	isula rm -f one > /dev/null
}
main(){
	test_001
	test_002
	test_003
	test_005
	test_006
	test_007
	test_008
}

main

