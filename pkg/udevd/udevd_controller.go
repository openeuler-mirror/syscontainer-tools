// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: udevd controller
// Author: zhangwei
// Create: 2018-01-18

package udevd

import (
	"fmt"
	"os"
	"path/filepath"

	hconfig "isula.org/isulad-tools/config"
	"golang.org/x/sys/unix"
)

var (
	programe   = "/lib/udev/isulad-tools_wrapper"
	lockFile   = "udevd_config_locker"
	configPath = "/etc/udev/rules.d/99-isulad-tools.rules"
)

// Rule defines an udev rule which used to capture the partition udev event
type Rule struct {
	Name       string
	CtrDevName string
	Container  string
}

// ToUdevRuleString will format the Rule structure to udev rule
func (r *Rule) ToUdevRuleString() string {
	return fmt.Sprintf("KERNEL==\"%s*\",ACTION==\"add|remove\", ENV{DEVTYPE}==\"partition\", SUBSYSTEM==\"block\", RUN{program}+=\"%s $env{ACTION} %s $name %s %s\"",
		filepath.Base(r.Name), programe, r.TrimContainerID(), r.CtrDevName, filepath.Base(r.Name))
}

// TrimContainerID will return container id for short
func (r *Rule) TrimContainerID() string {
	return r.Container[:8]
}

// Controller is the interface which to manage udev rules
type Controller interface {
	Lock() error
	Unlock() error
	LoadRules() error
	AddRule(r *Rule)
	RemoveRule(r *Rule)
	ToDisk() error
}

// NewUdevdController will return an UdevController interface
func NewUdevdController() Controller {
	using, err := usingUdevd()
	if err != nil {
		return &udevdController{useUdevd: false}
	}
	return &udevdController{
		dirty:      false,
		useUdevd:   using,
		configFile: configPath,
		lockFile:   filepath.Join(hconfig.IsuladToolsDir, lockFile),
	}
}

type udevdController struct {
	useUdevd   bool
	configFile string
	lockFile   string
	rules      []*Rule
	dirty      bool
	lock       *os.File
}

// Lock uses filelock to lock the udev rule file.
// to make sure only one process could access the resource.
func (sc *udevdController) Lock() error {
	if !sc.useUdevd {
		return nil
	}
	f, err := os.OpenFile(sc.lockFile, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	// FileLock will be released at 3 conditions:
	//  1. process to unlock manully.
	//  2. Close the opened fd.
	//  3. process died without call unlock. kernel will close the file and release the lock.
	// LOCK_EX means only one process could lock it at one time.
	// LOCK_NB is not set, using block mode.
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return err
	}
	sc.lock = f
	return nil
}

// Unlock will release the file lock
func (sc *udevdController) Unlock() error {
	if !sc.useUdevd || sc.lock == nil {
		return nil
	}
	defer sc.lock.Close()
	return unix.Flock(int(sc.lock.Fd()), unix.LOCK_UN)
}

// LoadRules loads the udev rules from rule config file
func (sc *udevdController) LoadRules() error {
	if !sc.useUdevd {
		return nil
	}

	rules, err := loadRules(sc.configFile)
	if err != nil {
		return err
	}
	sc.rules = rules
	return nil
}

// AddRule will add a rule to manager in memory only
func (sc *udevdController) AddRule(r *Rule) {
	if !sc.useUdevd {
		return
	}

	for _, rule := range sc.rules {
		if r.Name == rule.Name && r.CtrDevName == rule.CtrDevName && r.TrimContainerID() == rule.TrimContainerID() {
			return
		}
	}
	sc.dirty = true
	sc.rules = append(sc.rules, r)
	return
}

// RemoveRule will add a rule to manager in memory only
func (sc *udevdController) RemoveRule(r *Rule) {
	if !sc.useUdevd {
		return
	}

	for index, rule := range sc.rules {
		if r.Name == rule.Name && r.CtrDevName == rule.CtrDevName && r.TrimContainerID() == rule.TrimContainerID() {
			sc.dirty = true
			sc.rules = append(sc.rules[:index], sc.rules[index+1:]...)
			return
		}
	}
	return
}

// ToDisk will save the rules to udev rule config file
func (sc *udevdController) ToDisk() error {
	if !sc.useUdevd || !sc.dirty {
		return nil
	}

	if err := saveRules(sc.configFile, sc.rules); err != nil {
		return err
	}
	return reloadConfig()
}
