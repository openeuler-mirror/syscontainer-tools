// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: container config operation
// Author: zhangwei
// Create: 2018-01-18

package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var (
	// deviceHookLock is the default isulad container file lock name
	deviceHookLock = ".device_hook.lock"

	// restrictedNameChars collects the characters allowed to represent a name, normally used to validate container and volume names.
	restrictedNameChars = `[a-zA-Z0-9][a-zA-Z0-9_.-]`

	// restrictedNamePattern is a regular expression to validate names against the collection of restricted characters.
	restrictedNamePattern = regexp.MustCompile(`^/?` + restrictedNameChars + `+$`)
)

// Container is a structure which contains the basic config of isulad containers
type Container struct {
	pid           int
	containerID   string
	containerPath string
	name          string
	spec          *specs.Spec
	lock          *os.File
}

// New will create a container via a container name
func New(name string) (*Container, error) {
	if !restrictedNamePattern.MatchString(name) {
		return nil, fmt.Errorf("Invalid container name (%s), only %s are allowed", name, restrictedNameChars)
	}

	graphDriverPath, err := getIsuladGraphDriverPath()

	var id, storagePath string
	var pid int
	var spec *specs.Spec
	storagePath = filepath.Join(graphDriverPath, "engines", "lcr")
	id, err = getIsuladContainerID(name)
	if err != nil {
		return nil, err
	}
	pid, err = getIsuladContainerPid(name)
	if err != nil {
		return nil, err
	}
	spec, err = getIsuladContainerSpec(id)
	if err != nil {
		logrus.Warnf("fail to get isulad container %v spec: %v", id, err)
	}

	container := &Container{
		pid:           pid,
		name:          name,
		containerID:   id,
		spec:          spec,
		containerPath: filepath.Join(storagePath, id),
	}
	return container, nil
}

// Pid returns the pid of the container
func (c *Container) Pid() int {
	return c.pid
}

// ContainerID returns the container id
func (c *Container) ContainerID() string {
	return c.containerID
}

// Name returns the container name input by user.
func (c *Container) Name() string {
	return c.name
}

// ContainerPath returns the container config storage path.
func (c *Container) ContainerPath() string {
	return c.containerPath
}

// NetNsPath returns the net namespace path of the container
func (c *Container) NetNsPath() string {
	return fmt.Sprintf("/proc/%d/ns/net", c.pid)
}

// Lock uses file lock to lock the container
// to make sure only one handler could access the container resource
func (c *Container) Lock() error {
	fileName := filepath.Join(c.ContainerPath(), deviceHookLock)
	f, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0600)
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
	c.lock = f
	return nil
}

// Unlock will release the file lock
func (c *Container) Unlock() error {
	defer c.lock.Close()
	return unix.Flock(int(c.lock.Fd()), unix.LOCK_UN)
}

// GetCgroupPath returns the cgroup-parent segment of the container.
// For isulad container, it is a configurable segment.
func (c *Container) GetCgroupPath() (string, error) {
	cmd := exec.Command("isula", "inspect", "-f", "{{json .HostConfig.CgroupParent}}", c.name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %v", string(out), err)
	}
	cgroupPath := strings.Trim(string(out), "\n")
    if len(cgroupPath) >= 2 {
        cgroupPath = cgroupPath[1 : len(cgroupPath)-1]
    }
	if cgroupPath == "" {
		// by default, the cgroup path is "/lxc/<id>"
		cgroupPath = "/lxc"
	}
	cgroupPath = filepath.Join(cgroupPath, c.containerID)
	return cgroupPath, nil

}

// GetSpec get container spec
func (c *Container) GetSpec() *specs.Spec {
	return c.spec
}

// getIsuladContainerID returns the isulad container ID via the container name
func getIsuladContainerID(name string) (string, error) {
	cmd := exec.Command("isula", "inspect", "-f", "{{json .Id}}", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %v", string(out), err)
	}
	return strings.Trim(strings.Trim(string(out), "\n"), "\""), nil
}

// getIsuladContainerPid returns the isulad container process id via the container name
func getIsuladContainerPid(name string) (int, error) {
	cmd := exec.Command("isula", "inspect", "-f", "{{json .State.Pid}}", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return -1, fmt.Errorf("%s: %v", string(out), err)
	}
	strPid := strings.Trim(string(out), "\n")
	pid, err := strconv.Atoi(strPid)
	if err != nil {
		return -1, fmt.Errorf("failed to convert %q to int: %v", strPid, err)
	}
	return pid, nil
}

func getIsuladContainerSpec(id string) (spec *specs.Spec, err error) {
	graphDriverPath, err := getIsuladGraphDriverPath()
	if err != nil {
		return nil, err
	}
	configPath := fmt.Sprintf("%s/engines/lcr/%s/config.json", graphDriverPath, id)
	config, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s not found", configPath)
		}
		return nil, err
	}
	defer func() {
		if config != nil {
			config.Close()
		}
	}()
	if err := json.NewDecoder(config).Decode(&spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func getIsuladGraphDriverPath() (string, error) {
	cmd := exec.Command("isula", "info")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Exec isula info failed: %v", err)
	}
	// Find "iSulad Root Dir: /xx/xx" line. and out is still has the rest characters.
	if index := strings.Index(string(out), "iSulad Root Dir:"); index != -1 {
		// Split to array, and the first line is the "iSulad Root Dir"
		arr := strings.Split(string(out)[index:], "\n")
		// Split to find "  /xxx/xxx"
		array := strings.Split(arr[0], ":")
		if len(array) > 1 {
			// Trim all the spaces.
			rootdir := strings.Trim(array[1], " ")
			return rootdir, nil
		}
	}
	return "", fmt.Errorf("Faild to parse isula info, no \"iSulad Root Dir:\" found")
}

// SetContainerPath set container path
func (c *Container) SetContainerPath(path string) {
	c.containerPath = path
	return
}

// CheckPidExist check pid exist or not
func (c *Container) CheckPidExist() bool {
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", c.Pid())); err != nil {
		return false
	}
	return true
}
