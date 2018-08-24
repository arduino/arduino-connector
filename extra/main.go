//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2018  Arduino AG (http://www.arduino.cc/)
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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DylibMap struct {
	Name     string
	Provides []string
	URL      string
	Help     string
}

func main() {
	var v = make([]DylibMap, 0, len(os.Args[1:]))

	for _, arg := range os.Args[1:] {
		var lib DylibMap
		lib.Name = filepath.Base(arg)

		filepath.Walk(arg, func(path string, f os.FileInfo, err error) error {
			if strings.Contains(f.Name(), ".so") {
				lib.Provides = append(lib.Provides, f.Name())
			}
			return nil
		})
		lib.Help = "Please install " + lib.Name + " library from Intel website http://intel.com/"
		v = append(v, lib)
	}
	bytes, err := json.Marshal(v)
	if err == nil {
		fmt.Println(string(bytes))
	}
}
