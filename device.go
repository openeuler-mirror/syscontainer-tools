// Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
// isulad-tools is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//    http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.
// Description: device commands
// Author: zhangwei
// Create: 2018-01-18

// go base main package
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	hconfig "isula.org/isulad-tools/config"
	"isula.org/isulad-tools/container"
	"isula.org/isulad-tools/libdevice"
	"isula.org/isulad-tools/types"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var addDevCommand = cli.Command{
	Name:      "add-device",
	Usage:     "add one or more host devices to container",
	ArgsUsage: `<container_id> hostdevice[:containerdevice][:permission] [hostdevice[:containerdevice][:permission] ...]`,
	Description: `You can add mutiple host devices to container.
The program will error out when the host device is not a device or the container device already exists.`,
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "blkio-weight-device",
			Usage: "Set Block IO weight (relative device weight, between 10 and 1000)",
		},
		cli.StringSliceFlag{
			Name:  "device-read-bps",
			Usage: "Limit read rate (bytes per second) from a device",
		},
		cli.StringSliceFlag{
			Name:  "device-read-iops",
			Usage: "Limit read rate (IO per second) from a device",
		},
		cli.StringSliceFlag{
			Name:  "device-write-bps",
			Usage: "Limit write rate (bytes per second) to a device",
		},
		cli.StringSliceFlag{
			Name:  "device-write-iops",
			Usage: "Limit write rate (IO per second) to a device",
		},
		cli.BoolFlag{
			Name:  "follow-partition",
			Usage: "If disk is a base device, add all the sub partitions to container",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "If device exists in container, will cover the old file.",
		},
		cli.BoolFlag{
			Name:  "update-config-only",
			Usage: "If this flag is set, will not add device to container but update config only",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 2 {
			fatalf("%s: %q requires a minimum of 2 args", os.Args[0], context.Command.Name)
		}

		blkioWeight, err := libdevice.ParseAddDeviceBlkioWeight(context.StringSlice("blkio-weight-device"))
		if err != nil {
			fatal(err)
		}

		readBps, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-read-bps"))
		if err != nil {
			fatal(err)
		}
		writeBps, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-write-bps"))
		if err != nil {
			fatal(err)
		}
		readIOPS, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-read-iops"))
		if err != nil {
			fatal(err)
		}
		writeIOPS, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-write-iops"))
		if err != nil {
			fatal(err)
		}

		devices, err := getDevices(context)
		if err != nil {
			fatal(err)
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		if err := setDevicesPath(c, devices); err != nil {
			fatal(err)
		}

		opts := &types.AddDeviceOptions{
			Force:            context.Bool("force"),
			UpdateConfigOnly: context.Bool("update-config-only"),
			ReadBps:          readBps,
			WriteBps:         writeBps,
			ReadIOPS:         readIOPS,
			WriteIOPS:        writeIOPS,
			BlkioWeight:      blkioWeight,
		}

		// handle add device here
		if err = libdevice.AddDevice(c, devices, opts); err != nil {
			fatalf("Failed to add device: %v", err)
		}

		logrus.Infof("add device to container %q successfully", name)
		return
	},
}

var rmDevCommand = cli.Command{
	Name:      "remove-device",
	Usage:     "remove one or more devices from container",
	ArgsUsage: `<container_id> hostdevice[:containerdevice] [hostdevice[:containerdevice] ...]`,
	Description: `You can remove mutiple host devices from container.
You can assign hostdevice an empty value, though either of hostdevice and containerdevice should be assigned.
The program will error out when the container device does not exist.`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "follow-partition",
			Usage: "If disk is a base device, will remove all the sub partitions from container",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 2 {
			fatalf("%s: %q requires a minimum of 2 args", os.Args[0], context.Command.Name)
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		devices, err := getMappings(context)
		if err != nil {
			fatal(err)
		}
		if err := setDevicesPath(c, devices); err != nil {
			fatal(err)
		}

		// handle remove device here
		if err = libdevice.RemoveDevice(c, devices, context.Bool("follow-partition")); err != nil {
			fatalf("Failed to remove device: %v", err)
		}
		logrus.Infof("remove device from container %q successfully", name)
		return
	},
}

var listDevCommand = cli.Command{
	Name:      "list-device",
	Usage:     "list all devices in container",
	ArgsUsage: `<container_id>`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "pretty, p",
			Usage: "If this flag is set, list pathes in pretty json form",
		},
		cli.BoolFlag{
			Name:  "sub-partition",
			Usage: "If disk is a base device, list all the sub partitions by the base disk",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q must accept a container-id", os.Args[0], context.Command.Name)
		}
		if context.NArg() > 1 {
			fatalf("Don't put container-id in the middle of options")
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		allDevices, majorDevices, err := libdevice.ListDevice(c)
		if err != nil {
			fatalf("Failed to list device in container: %v", err)
		}

		var outputDevices []*hconfig.DeviceMapping

		if context.Bool("sub-partition") {
			outputDevices = allDevices
		} else {
			outputDevices = majorDevices
		}

		if outputDevices == nil || len(outputDevices) == 0 {
			logrus.Infof("list device in container %q successfully", name)
			return
		}

		devicesData, err := json.Marshal(outputDevices)
		if err != nil {
			fatalf("failed to Marshal device config: %v", err)
		}
		devicesBuffer := new(bytes.Buffer)
		if _, err = devicesBuffer.Write(devicesData); err != nil {
			fatalf("Buffer Write error %v", err)
		}

		if context.Bool("pretty") {
			devicesBuffer.Truncate(0)
			if json.Indent(devicesBuffer, devicesData, "", "\t") != nil {
				fatalf("failed to Indent device data: %v", err)
			}
		}

		if _, err = devicesBuffer.WriteString("\n"); err != nil {
			fatalf("Buffer WriteString error %v", err)
		}

		if _, err = os.Stdout.Write(devicesBuffer.Bytes()); err != nil {
			logrus.Errorf("os.Stdout.Write error : %v", err)
		}
		logrus.Infof("list devices in container %q successfully", name)
	},
}

var updateDevCommand = cli.Command{
	Name:        "update-device",
	Usage:       "update configuration of device",
	ArgsUsage:   `<container_id>`,
	Description: `You can update configuration of container devices.`,
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "device-read-bps",
			Usage: "Limit read rate (bytes per second) from a device",
		},
		cli.StringSliceFlag{
			Name:  "device-read-iops",
			Usage: "Limit read rate (IO per second) from a device",
		},
		cli.StringSliceFlag{
			Name:  "device-write-bps",
			Usage: "Limit write rate (bytes per second) to a device",
		},
		cli.StringSliceFlag{
			Name:  "device-write-iops",
			Usage: "Limit write rate (IO per second) to a device",
		},
	},
	Action: func(context *cli.Context) {
		if context.NArg() < 1 {
			fatalf("%s: %q requires a minimum of 1 args", os.Args[0], context.Command.Name)
		}

		readBps, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-read-bps"))
		if err != nil {
			fatal(err)
		}
		writeBps, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-write-bps"))
		if err != nil {
			fatal(err)
		}
		readIOPS, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-read-iops"))
		if err != nil {
			fatal(err)
		}
		writeIOPS, err := libdevice.ParseAddDeviceQosOption(context.StringSlice("device-write-iops"))
		if err != nil {
			fatal(err)
		}

		if len(readBps) == 0 && len(writeBps) == 0 && len(readIOPS) == 0 && len(writeIOPS) == 0 {
			fatalf("update device should specify at least one device QOS configuration")
		}

		name := context.Args()[0]
		c, err := container.New(name)
		if err != nil {
			fatal(err)
		}

		opts := &types.AddDeviceOptions{
			ReadBps:   readBps,
			WriteBps:  writeBps,
			ReadIOPS:  readIOPS,
			WriteIOPS: writeIOPS,
		}

		// handle add device here
		if err = libdevice.UpdateDevice(c, opts); err != nil {
			fatalf("Failed to update device: %v", err)
		}

		logrus.Infof("update device configure in container %q successfully", name)
		return
	},
}

func getDevices(context *cli.Context) ([]*types.Device, error) {
	var devices []*types.Device
	followPartition := context.Bool("follow-partition")
	for k := 1; k < context.NArg(); k++ {
		v := context.Args()[k]
		device, err := libdevice.ParseDevice(context.Args()[k])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse device: %s, %v", v, err)
		}

		if device.Type == "c" {
			if followPartition {
				return nil, fmt.Errorf("Char device %s not support follow partition", v)
			}
			devices = append(devices, device)
			continue
		}

		basedev, err := types.GetBaseDevName(device.PathOnHost)
		if err != nil {
			return nil, err
		}
		devType, err := types.GetDeviceType(device.PathOnHost)
		if err != nil {
			return nil, err
		}
		if devType != "lvm" {
			device.Parent = basedev
		}

		devices = append(devices, device)

		if followPartition && devType == "disk" {
			// Add sub-partition here
			subDevices := libdevice.FindSubPartition(device)
			for _, dev := range subDevices {
				found := false
				for _, eDev := range devices {
					if dev.PathOnHost == eDev.PathOnHost {
						found = true
						break
					}
				}
				if !found {
					devices = append(devices, dev)
				}
			}
		}
	}
	return devices, nil
}

func getMappings(context *cli.Context) ([]*types.Device, error) {
	var devices []*types.Device
	for k := 1; k < context.NArg(); k++ {
		v := context.Args()[k]
		device, err := libdevice.ParseMapping(context.Args()[k])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse device mapping: %s, %v", v, err)
		}
		devices = append(devices, device)
	}
	return devices, nil

}

func setDevicesPath(c *container.Container, devices []*types.Device) error {
	if err := c.Lock(); err != nil {
		return err
	}
	defer c.Unlock()
	config, err := hconfig.NewContainerConfig(c)
	if err != nil {
		return err
	}
	for _, dev := range devices {
		if index := config.DeviceIndexInArray(dev); index != -1 {
			found := config.GetAllDevices()[index]
			dev.Path = found.PathInContainer
			dev.PathOnHost = found.PathOnHost
		}
		libdevice.SetDefaultPath(dev)
	}
	return nil
}
