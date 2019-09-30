// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: udevd rules
// Author: zhangwei
// Create: 2018-01-18

package udevd

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func usingUdevd() (bool, error) {
	return true, nil
}

func reloadConfig() error {
	_, err := exec.Command("udevadm", "control", "--reload").CombinedOutput()
	return err
}

func saveRules(path string, rules []*Rule) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}

	f.WriteString("## This File is auto-generated by isulad-tools.\n")
	f.WriteString("## DO   NOT  EDIT   IT\n\n")
	for _, r := range rules {
		if _, err := f.WriteString(fmt.Sprintf("%s\n", r.ToUdevRuleString())); err != nil {
			logrus.Errorf("f.WriteString err: %s", err)
		}
	}

	if err := f.Sync(); err != nil {
		logrus.Errorf("f.WriteString err: %s", err)
		return err
	}
	return nil
}

func loadRules(path string) ([]*Rule, error) {
	var rules []*Rule
	f, err := os.Open(path)
	if err != nil {
		// if non-existing, just return empty rules array
		if os.IsNotExist(err) {
			return rules, nil
		}
		return nil, err
	}
	defer f.Close()

	logrus.Infof("Start load rules from path: %s", path)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimLeft(scanner.Text(), " ")

		// ignore the comment line
		if strings.HasPrefix(text, "#") {
			continue
		}

		array := strings.Split(text, ",")

		var (
			name        string
			ctrDevName  string
			containerID string
		)

		for _, tag := range array {
			tag = strings.TrimLeft(tag, " ")
			tag = strings.TrimRight(tag, " ")

			// Parse device name from 'KERNEL' segment
			if strings.HasPrefix(tag, "KERNEL") {
				ar := strings.Split(tag, "\"")
				if len(ar) >= 2 { // Minimum value for get split tag
					sub := ar[1]
					name = sub[:len(sub)-1]
				}
			}

			// Parse Container ID, major, minor number from 'RUN' command
			if strings.HasPrefix(tag, "RUN") {
				ar := strings.Split(tag, "\"")
				if len(ar) >= 2 { // Minimum value for get split tag
					sub := ar[1]
					cmdArray := strings.Split(sub, " ")
					if len(cmdArray) >= 6 { // Minimum value for get split sub
						containerID = cmdArray[2]
						ctrDevName = cmdArray[4]
					}
				}
			}
		}
		// TODO: failed to parse some rules, ignore it??
		if name == "" || containerID == "" || ctrDevName == "" {
			continue
		}
		rules = append(rules, &Rule{
			Name:       filepath.Join("/dev", name),
			CtrDevName: ctrDevName,
			Container:  containerID,
		})
	}
	logrus.Infof("Finish load rules from path: %s", path)
	return rules, nil
}
