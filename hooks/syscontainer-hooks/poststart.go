// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: poststart hook
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/sirupsen/logrus"
)

// prestartHook is the main logic of device hook
func poststartHook(data *hookData, withRelabel bool) error {
	var actions []HookAction
	actions = []HookAction{}
	for _, ac := range actions {
		if err := ac(data.state, data.hookConfig, data.spec); err != nil {
			logrus.Errorf("Failed with err: %v", err)
			return err
		}
	}
	return nil
}
