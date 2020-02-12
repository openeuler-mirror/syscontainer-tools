// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: parse device major and minor number
// Author: zhangwei
// Create: 2018-01-18

package libdevice

/*

This code provides support for manipulating linux device numbers.  It should be replaced by normal syscall functions once http://code.google.com/p/go/issues/detail?id=8106 is solved.

You can read what they are here:

 - http://www.makelinux.net/ldd3/chp-3-sect-2
 - http://www.linux-tutorial.info/modules.php?name=MContent&pageid=94

Note! These are NOT the same as the MAJOR(dev_t device);, MINOR(dev_t device); and MKDEV(int major, int minor); functions as defined in <linux/kdev_t.h> as the representation of device numbers used by go is different than the one used internally to the kernel! - https://github.com/torvalds/linux/blob/master/include/linux/kdev_t.h#L9

*/

// Major returns the major number of a device
func Major(devNumber int) int64 {
	return int64((devNumber >> 8) & 0xfff)
}

// Minor returns the minor number of a device
func Minor(devNumber int) int64 {
	return int64((devNumber & 0xff) | ((devNumber >> 12) & 0xfff00))
}
