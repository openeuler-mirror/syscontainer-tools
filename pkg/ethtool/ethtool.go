// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: ethtool for network interface
// Author: zhangwei
// Create: 2018-01-18

package ethtool

/*
#cgo LDFLAGS: -lsecurec
#include <stdlib.h>
#include "ethtool.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

// Ethtool ethtool interface
type Ethtool interface {
	SetNetDeviceTSO(on bool) error
	SetNetDeviceTX(on bool) error
	SetNetDeviceSG(on bool) error
	Close() error
}

type ethtool struct {
	fd   int
	name string
	sync.Mutex
}

// NewEthtool init ethtool
func NewEthtool(name string) (Ethtool, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		fd, err = syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_GENERIC)
	}
	if err != nil {
		return nil, err
	}
	return &ethtool{fd: fd, name: name}, nil
}

func (etool *ethtool) SetNetDeviceTSO(on bool) error {
	etool.Lock()
	defer etool.Unlock()

	var cOn C.int
	if on {
		cOn = 1
	}

	cName := C.CString(etool.name)
	ret := C.setNetDeviceTSO(C.int(etool.fd), cName, cOn)
	C.free(unsafe.Pointer(cName))
	if ret != 0 {
		return fmt.Errorf("setNetDeviceTso return error: %d", ret)
	}
	return nil
}

func (etool *ethtool) SetNetDeviceTX(on bool) error {
	etool.Lock()
	defer etool.Unlock()

	var cOn C.int
	if on {
		cOn = 1
	}

	cName := C.CString(etool.name)
	ret := C.setNetDeviceTX(C.int(etool.fd), cName, cOn)
	C.free(unsafe.Pointer(cName))
	if ret != 0 {
		return fmt.Errorf("setNetDeviceTso return error: %d", ret)
	}
	return nil
}

func (etool *ethtool) SetNetDeviceSG(on bool) error {
	etool.Lock()
	defer etool.Unlock()

	var cOn C.int
	if on {
		cOn = 1
	}

	cName := C.CString(etool.name)
	ret := C.setNetDeviceSG(C.int(etool.fd), cName, cOn)
	C.free(unsafe.Pointer(cName))
	if ret != 0 {
		return fmt.Errorf("setNetDeviceTso return error: %d", ret)
	}
	return nil
}

func (etool *ethtool) Close() error {
	return syscall.Close(etool.fd)
}
