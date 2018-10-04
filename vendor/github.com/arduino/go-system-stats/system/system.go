//
//  This file is part of go-system-stats library
//
//  Copyright (C) 2018  Arduino AG (http://www.arduino.cc/)
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

package system

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

// GetUptime returns the system uptime
func GetUptime() (time.Duration, error) {
	data, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("Reading /proc/uptime: %s", err)
	}
	timeString := strings.Fields(string(data))[0]
	t, err := strconv.ParseFloat(timeString, 64)
	if err != nil {
		return 0, fmt.Errorf("Parsing /proc/uptime: %s", err)
	}
	res := time.Millisecond * time.Duration(t*1000)
	return res, nil
}
