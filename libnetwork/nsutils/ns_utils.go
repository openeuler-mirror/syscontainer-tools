// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: netns invoke
// Author: zhangwei
// Create: 2018-01-18

package nsutils

import (
	"fmt"
	"runtime"

	"github.com/vishvananda/netns"
)

// NsInvoke function is used for setting network outside/inside the container/netns
// prefunc is called in the host, and postfunc is used in container
func NsInvoke(path string, prefunc func(nsFD int) error, postfunc func(callerFD int) error) error {
	initns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed get network namespace %v", err)
	}
	defer initns.Close()

	ns, err := netns.GetFromPath(path)
	if err != nil {
		return fmt.Errorf("failed get network namespace %s: %v", path, err)
	}
	defer ns.Close()

	// Invoked before the namespace switch happens but after the namespace file
	// handle is obtained.
	if err := prefunc(int(ns)); err != nil {
		return fmt.Errorf("failed in prefunc: %v", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err = netns.Set(ns); err != nil {
		return err
	}
	// Invoked after the namespace switch.
	err = postfunc(int(initns))
	if err1 := netns.Set(initns); err1 != nil {
		return fmt.Errorf("failed to set to initial namespace: %v: %v", err1, err)
	}

	return err
}
