// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: device and bind type
// Author: zhangwei
// Create: 2018-01-18

package types

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/sirupsen/logrus"
)

const (
	wildcard = -1
	majorNum = 8  // the device number of major
	minorNum = 12 // the device number of minor
)

// ErrMsg is a structure used by parent and child processes to transfer error messages
type ErrMsg struct {
	Error string
}

// AddDeviceMsg is a parent and child message, used to transfer 'add device' operation
type AddDeviceMsg struct {
	Force  bool
	Device *Device
}

// Bind is a parent and child message, used to transfer bind operation
type Bind struct {
	HostPath      string // Path on Host relative path, (based on entry point.)
	IsDir         bool   // Path is Directory?
	ResolvPath    string // Relative path of Mountpoint
	ContainerPath string // Path in Container
	MountOption   string // Bind Mount options, to dest path.
	UID           int    // User ID
	GID           int    // Group ID
}

// ToString returns the storage format string of the bind in device hook config file
func (bind *Bind) ToString() string {
	return fmt.Sprintf("%s:%s:%s", bind.HostPath, bind.ContainerPath, bind.MountOption)
}

// Device is the device abstract structure used by the entire workspace
type Device struct {
	Type        string      // Device type: c or b
	Path        string      // Path in container.
	PathOnHost  string      // Path on Host.
	Major       int64       // Major number of device
	Minor       int64       // Minor number of device
	Permissions string      // Permissions which user input.
	FileMode    os.FileMode // File Mode,
	UID         uint32      // User ID
	GID         uint32      // Group ID
	Allow       bool        // Used to differ add or remove
	Parent      string      // Parent device name(pathonhost)
}

// Qos is the device Qos structure
type Qos struct {
	Major int64  `json:"major"`
	Minor int64  `json:"minor"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// AddDeviceOptions defines the optsions for add device operation
type AddDeviceOptions struct {
	ReadBps          []*Qos
	WriteBps         []*Qos
	ReadIOPS         []*Qos
	WriteIOPS        []*Qos
	BlkioWeight      []*Qos
	Force            bool
	UpdateConfigOnly bool
}

func (q Qos) String() string {
	return fmt.Sprintf("%d:%d %s", q.Major, q.Minor, q.Value)
}

// GetCfqAbility get cfq ability or not
func (q Qos) GetCfqAbility() (bool, error) {
	q.Path = filepath.Clean(q.Path)
	Cfq, err := q.ReadCFQ(q.Path)
	if err != nil {
		return false, err
	}
	return Cfq, err
}

// ReadCFQ read cfq value
func (q Qos) ReadCFQ(devName string) (bool, error) {
	path := fmt.Sprintf("/sys/block/%s/queue/scheduler", filepath.Base(devName))

	cfqFile, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stdout, "fail to open cfq file:%s", path)
		return false, err
	}
	defer cfqFile.Close()

	buff := bufio.NewReader(cfqFile)

	line, _, err := buff.ReadLine()
	if err != nil && err != io.EOF {
		return false, err
	}

	if strings.Contains(string(line), "cfq") || strings.Contains(string(line), "bfq") {
		return true, nil
	}
	return false, nil

}

// String returns the device string which stores in config file
func (d *Device) String() string {
	return fmt.Sprintf("%s:%s:%s", d.PathOnHost, d.Path, d.Permissions)
}

// CgroupString returns the cgroup string of a device
func (d *Device) CgroupString() string {
	// Agreement with Product:
	//  we do not care about sub logic block device, they will take care of it.
	return fmt.Sprintf("%s %s:%s %s", d.Type, deviceNumberString(d.Major), deviceNumberString(d.Minor), d.Permissions)
}

// GetBaseDevName returns base device of a partition
func GetBaseDevName(deviceName string) (string, error) {
	cmd := exec.Command("lsblk", "-n", "-p", "-r", "-s", "-o", "NAME", deviceName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to lsblk %s : %v", string(out), err)
	}
	rawString := strings.Split(string(out), "\n")
	if len(rawString) <= 1 {
		return "", nil
	}
	return rawString[1], nil
}

// GetDeviceType get device type
func GetDeviceType(devName string) (string, error) {
	cmd := exec.Command("lsblk", "-n", "-d", "-o", "TYPE", devName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to lsblk %s : %v", string(out), err)
	}
	devType := strings.Trim(string(out), "\n")
	logrus.Debugf("%s type is %s\n", devName, devType)
	return devType, nil
}

// Mkdev returns the device number in int format
func (d *Device) Mkdev() int {
	return int((d.Major << majorNum) | (d.Minor & 0xff) | ((d.Minor & 0xfff00) << minorNum))
}

// deviceNumberString returns the device number by string
func deviceNumberString(number int64) string {
	if number == wildcard {
		return "*"
	}
	return fmt.Sprint(number)
}
