// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: parse device option
// Author: zhangwei
// Create: 2018-01-18

package libdevice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"isula.org/syscontainer-tools/types"
)

const (
	// KB equals 1024Byte
	KB float64 = 1024
	// MB equals 1024KB
	MB float64 = 1024 * KB
	// GB equals 1024MB
	GB float64 = 1024 * MB
	// TB equals 1024GB
	TB float64 = 1024 * GB
	// PB equals 1024TB
	PB float64 = 1024 * TB
)

var (
	unitMap = map[string]float64{"k": KB, "m": MB, "g": GB, "t": TB, "p": PB}
	reg     = regexp.MustCompile(`^(\d+(?:\.\d+)*)?([kKmMgGtTpP])?[bB]?$`)
)

func parseSize(sizeString string) (int64, error) {
	matches := reg.FindStringSubmatch(sizeString)
	if len(matches) != 3 { // valid values for parse string len
		return -1, fmt.Errorf("Invalid size: '%s'", sizeString)
	}
	size, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return -1, err
	}
	if unit, ok := unitMap[strings.ToLower(matches[2])]; ok {
		size *= unit
	}
	return int64(size), nil
}

// ParseAddDeviceQosOption parse the inpt blkio options into []types.Qos for add device command.
func ParseAddDeviceQosOption(vals []string) ([]*types.Qos, error) {
	var devQos []*types.Qos

	for _, val := range vals {
		split := strings.SplitN(val, ":", 2)
		if len(split) < 2 { // the number of substrings to return
			return nil, fmt.Errorf("Bad format: %s", val)
		}
		dev, err := ParseDevice(split[0])
		if err != nil {
			return nil, err
		}

		if dev.Type == "c" {
			return nil, fmt.Errorf("cannot set Qos of a char device")
		}
		devType, err := types.GetDeviceType(dev.PathOnHost)
		if err != nil {
			return nil, err
		}
		if devType == "part" {
			return nil, fmt.Errorf("cannot set Qos of a child device")
		}
		rate, err := parseSize(split[1])
		if err != nil || rate < 0 {
			return nil, fmt.Errorf("Invalid rate for device: %s. The correct format is <device-path>:<number>[<unit>]. Number must be a positive integer. Unit is optional and can be kb, mb, or gb", val)
		}
		devQos = append(devQos, &types.Qos{
			Path:  split[0],
			Major: dev.Major,
			Minor: dev.Minor,
			Value: fmt.Sprintf("%d", rate),
		})
	}
	return devQos, nil
}

// ParseAddDeviceBlkioWeight parse the inpt blkio weight into []types.Qos for add device command.
func ParseAddDeviceBlkioWeight(vals []string) ([]*types.Qos, error) {
	var devQos []*types.Qos

	for _, val := range vals {
		split := strings.SplitN(val, ":", 2)
		if len(split) < 2 { // the number of substrings to return
			return nil, fmt.Errorf("Bad format: %s", val)
		}
		dev, err := ParseDevice(split[0])
		if err != nil {
			return nil, err
		}

		if dev.Type == "c" {
			return nil, fmt.Errorf("cannot set Qos of a char device")
		}
		devType, err := types.GetDeviceType(dev.PathOnHost)
		if err != nil {
			return nil, err
		}
		if devType == "part" {
			return nil, fmt.Errorf("cannot set Qos of a child device")
		}

		weight, err := strconv.ParseUint(split[1], 10, 0)
		if err != nil {
			return nil, fmt.Errorf("Invalid weight for device: %s", val)
		}
		if weight > 0 && (weight < 10 || weight > 1000) { // Invalid weight for device interval value
			return nil, fmt.Errorf("Invalid weight for device: %s", val)
		}
		devQos = append(devQos, &types.Qos{
			Path:  split[0],
			Major: dev.Major,
			Minor: dev.Minor,
			Value: fmt.Sprintf("%d", weight),
		})

	}
	return devQos, nil
}
