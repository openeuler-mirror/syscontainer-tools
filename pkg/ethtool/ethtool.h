/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2017-2019. All rights reserved.
 * isulad-tools is licensed under the Mulan PSL v1.
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
#pragma once

#ifndef __ETHTOOL_H
#define __ETHTOOL_H

extern int setNetDeviceTSO(int fd, char* name, int on);
extern int setNetDeviceTX(int fd, char* name, int on);
extern int setNetDeviceSG(int fd, char* name, int on);

#endif
