// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: hook utils
// Author: zhangwei
// Create: 2018-01-18

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// isula info may result in dead lock when start with restart policy
// try to get isulad root path with hook path
func getGraphDriverPath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	// get /var/lib/isulad from /var/lib/isulad/hooks/isulad-hooks
	dir := filepath.Dir(filepath.Dir(path))
	return dir, nil
}

// GetContainerStoragePath returns the isulad container storage path
func GetContainerStoragePath() (string, error) {
	// check graph driver here!
	graphDriverPath, err := getGraphDriverPath()
	if err != nil {
		return "", err
	}
	base := filepath.Join(graphDriverPath, "engines", "lcr")
	finfo, err := os.Stat(base)
	if err != nil {
		return "", err
	}
	if !finfo.IsDir() {
		return "", fmt.Errorf("Container Path:%s is not a directory", base)
	}
	return base, nil
}

// CompatHookState hook state compat with old version
type CompatHookState struct {
	configs.SpecState
	Bundle string `json:"bundlePath"`
}

// ParseHookState parses the config of HookState from isulad via stdin
func ParseHookState(reader io.Reader) (*configs.HookState, error) {
	// We expect configs.HookState as a json string in <stdin>
	stateBuf, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var state configs.HookState
	if err = json.Unmarshal(stateBuf, &state); err != nil {
		return nil, err
	}

	var compatStat CompatHookState
	if state.Bundle == "" {
		if err = json.Unmarshal(stateBuf, &compatStat); err != nil {
			return nil, err
		}
		if compatStat.Bundle == "" {
			return nil, fmt.Errorf("unmarshal hook state failed %s", stateBuf)
		}
		state.Bundle = compatStat.Bundle
	}

	return &state, nil
}

// LoadSpec will load the oci config of isulad container from disk.
func LoadSpec(configPath string) (spec *specs.Spec, err error) {
	config, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s not found", configPath)
		}
		return nil, err
	}
	defer config.Close()

	if err = json.NewDecoder(config).Decode(&spec); err != nil {
		return nil, err
	}
	return spec, nil
}

// LoadCompatSpec will load the oci config of isulad container from disk.
func LoadCompatSpec(configPath string) (spec *specs.CompatSpec, err error) {
	config, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s not found", configPath)
		}
		return nil, err
	}
	defer config.Close()

	if err = json.NewDecoder(config).Decode(&spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func hostIDFromMapping(containerID uint32, uMap []specs.LinuxIDMapping) int {
	if uMap != nil {
		for _, m := range uMap {
			if (containerID >= m.ContainerID) && (containerID <= (m.ContainerID + m.Size - 1)) {
				hostID := m.HostID + (containerID - m.ContainerID)
				return int(hostID)
			}
		}
	}
	return -1
}

// GetUIDGid get uid and gid from spec
func GetUIDGid(spec *specs.Spec) (int, int) {
	for _, namespace := range spec.Linux.Namespaces {
		if namespace.Type == specs.UserNamespace {
			return hostIDFromMapping(0, spec.Linux.UIDMappings), hostIDFromMapping(0, spec.Linux.GIDMappings)
		}
	}
	return -1, -1
}
