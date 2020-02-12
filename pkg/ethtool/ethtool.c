/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2017-2019. All rights reserved.
 * syscontainer-tools is licensed under the Mulan PSL v1.
 * You can use this software according to the terms and conditions of the Mulan PSL v1.
 * You may obtain a copy of Mulan PSL v1 at:
 *    http://license.coscl.org.cn/MulanPSL
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v1 for more details.
 * Author: zhangwentao
 * Create: 2017-07-19
 * Description: config net device tx/sg/tso on
 ******************************************************************************/
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <unistd.h>
#include <endian.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <linux/sockios.h>
#include <netinet/in.h>
#include <net/if.h>
#include <linux/netlink.h>
#include <linux/ethtool.h>

int setNetDeviceTSO(int fd, char* name, int on)
{
    struct ifreq ifr = {0};
    struct ethtool_value eval = {0};
    int ret;
    eval.data = on;
    eval.cmd = ETHTOOL_STSO;

    strncpy(ifr.ifr_name, name, sizeof(ifr.ifr_name) - 1);
    ifr.ifr_data = (void*)&eval;
    ret = ioctl(fd, SIOCETHTOOL, &ifr);
    if (ret < 0) {
        return -1;
    }
    return 0;
}

int setNetDeviceSG(int fd, char* name, int on)
{
    struct ifreq ifr = {0};
    struct ethtool_value eval = {0};
    int ret;
    eval.data = on;
    eval.cmd = ETHTOOL_SSG;

    strncpy(ifr.ifr_name, name, sizeof(ifr.ifr_name) - 1);
    ifr.ifr_data = (void*)&eval;
    ret = ioctl(fd, SIOCETHTOOL, &ifr);
    if (ret < 0) {
        return -1;
    }
    return 0;
}

int setNetDeviceTX(int fd, char* name, int on)
{
    struct ifreq ifr = {0};
    struct ethtool_value eval = {0};
    int ret;
    eval.data = on;
    eval.cmd = ETHTOOL_STXCSUM;

    strncpy(ifr.ifr_name, name, sizeof(ifr.ifr_name) - 1);
    ifr.ifr_data = (void*)&eval;
    ret = ioctl(fd, SIOCETHTOOL, &ifr);
    if (ret < 0) {
        return -1;
    }
    return 0;
}
