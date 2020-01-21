// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// syscontainer-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: common utils
// Author: zhangwei
// Create: 2018-01-18

package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	libctr_utils "github.com/opencontainers/runc/libcontainer/utils"
	"github.com/sirupsen/logrus"
)

// SyslogHook to send logs via syslog.
type syslogHook struct {
	logger *log.Logger
}

// Creates a hook to be added to an instance of logger. This is called with
// `hook, err := newSyslogHook("default", "udp", "localhost:514", syslog.LOG_DEBUG, "")`
// `if err == nil { log.Hooks.Add(hook) }`
func newSyslogHook(network, raddr string, priority syslog.Priority, tag string) (*syslogHook, error) {
	var logger *log.Logger
	var err error

	if network == "default" {
		logger, err = syslog.NewLogger(priority, log.Lshortfile)
		if err != nil {
			return nil, err
		}
		logger.SetPrefix(tag)
	} else {
		w, err := syslog.Dial(network, raddr, priority, "")
		if err != nil {
			return nil, err
		}
		logger = log.New(w, tag, log.Lshortfile)
	}

	return &syslogHook{logger}, err
}

func (hook *syslogHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	if err := hook.logger.Output(8, line); err != nil {
		logrus.Errorf("hook.logger.Output err: %s", err)
	}
	return nil
}

func (hook *syslogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

const (
	syslogUDPPrefix         = "udp://"
	syslogTCPPrefix         = "tcp://"
	syslogUnixSock          = "unix://"
	syslogDefaultUDPService = "localhost:541"
	syslogDefaultTCPService = "localhost:541"
)

// SyslogService is a structure which records the syslog service type and serivce address.
type SyslogService struct {
	Type string
	Addr string
}

// ParseSyslogService parses syslog service from input string
func ParseSyslogService(service string) (*SyslogService, error) {
	var serviceType, serviceAddr string

	if service == "" {
		serviceType = "default"
		serviceAddr = ""
	} else if strings.HasPrefix(service, syslogUDPPrefix) {
		serviceType = "udp"
		serviceAddr := service[len(syslogUDPPrefix):]
		if serviceAddr == "" {
			serviceAddr = syslogDefaultUDPService
		}
	} else if strings.HasPrefix(service, syslogTCPPrefix) {
		serviceType = "tcp"
		serviceAddr = service[len(syslogTCPPrefix):]
		if serviceAddr == "" {
			serviceAddr = syslogDefaultTCPService
		}
	} else if strings.HasPrefix(service, syslogUnixSock) {
		// syslog package will use empty string as network,
		// and syslog will lookup the unix socket on host, we do not care.
		serviceType = ""
		serviceAddr = service[len(syslogUnixSock):]
	} else {
		return nil, fmt.Errorf("Unspported syslog network: %s", service)
	}

	serv := &SyslogService{
		Type: serviceType,
		Addr: serviceAddr,
	}
	return serv, nil
}

// HookSyslog will hook syslog service to logrus
// syslog supports 4 kinds of service:
//     1. default socket: ""
//     2. unix socket:    "unix:///dev/log"
//     3. udp  port:	  "udp://localhost:541"
//     4. tcp  port:      "tcp://localhost:541"
//    by default, if we output to local syslog, use default will be fine.
// syslog Tag:
//   syslog will use tag to separate the output stream.
func HookSyslog(service, tag string) error {
	serv, err := ParseSyslogService(service)
	if err != nil {
		return err
	}

	hook, err := newSyslogHook(serv.Type, serv.Addr, syslog.LOG_INFO|syslog.LOG_USER, tag)
	if err != nil {
		return fmt.Errorf("Unable to connect to syslog daemon")
	}
	logrus.AddHook(hook)
	return nil
}

// NewPipe creates a pair of  new socket pipe.
func NewPipe() (parent, child *os.File, err error) {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), "parent"), os.NewFile(uintptr(fds[0]), "child"), nil
}

// WriteJSON write json data to io stream
func WriteJSON(w io.Writer, v interface{}) error {
	return libctr_utils.WriteJSON(w, v)
}

// RandomID returns a 8-bit ramdon string which read from rand.Reader first,
// and if failed, will use time stamp as random id
func RandomID() string {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		cur := time.Now()
		return fmt.Sprint(cur.UnixNano())
	}
	return hex.EncodeToString(id)[:8]
}

// RandomFile will find a non-existing file in given folder.
func RandomFile(folder string) string {
	path := ""
	for {
		id := RandomID()
		path = filepath.Join(folder, id)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
	}
	return path
}
