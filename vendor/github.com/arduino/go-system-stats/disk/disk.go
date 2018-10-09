//
//  This file is part of go-system-stats library
//
//  Copyright (C) 2017  Arduino AG (http://www.arduino.cc/)
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package disk

import (
	"fmt"
	"io/ioutil"
	"strings"
	"syscall"
)

// FSStats contains usage stats for a filesystem
type FSStats struct {
	Device         string
	Type           string
	MountPoint     string
	FreeSpace      uint64
	AvailableSpace uint64
	DiskSize       uint64
}

// GetStats returns usage stats for all mounted filesystems
func GetStats() ([]*FSStats, error) {
	data, err := ioutil.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil, fmt.Errorf("detecting mounted filesystems: %s", err)
	}
	mountInfos := strings.Split(string(data), "\n")

	res := []*FSStats{}
	for _, mountInfo := range mountInfos {
		fields := strings.Fields(mountInfo)
		if len(fields) < 5 {
			continue
		}

		mount := fields[4]

		statfs := &syscall.Statfs_t{}
		if err := syscall.Statfs(mount, statfs); err != nil {
			// ignore error
			continue
		}

		fs := &FSStats{
			MountPoint:     mount,
			Type:           fields[8],
			Device:         fields[9],
			FreeSpace:      statfs.Bfree * uint64(statfs.Bsize),
			AvailableSpace: statfs.Bavail * uint64(statfs.Bsize),
			DiskSize:       statfs.Blocks * uint64(statfs.Bsize),
		}
		res = append(res, fs)
	}

	return res, nil
}
