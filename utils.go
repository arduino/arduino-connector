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
	"os"
	"path/filepath"
	"strings"
)

func addIntelLibrariesToLdPath() {
	_, err := os.Stat("/opt/intel")
	if err == nil {
		//scan /opt/intel searching for sdks
		var extraPaths []string
		filepath.Walk("/opt/intel", func(path string, f os.FileInfo, err error) error {
			if strings.Contains(f.Name(), ".so") && !strings.Contains(path, "uninstall") {
				extraPaths = appendIfUnique(extraPaths, filepath.Dir(path))
			}
			return nil
		})
		os.Setenv("LD_LIBRARY_PATH", os.Getenv("LD_LIBRARY_PATH")+":"+strings.Join(extraPaths, ":"))
	}
}
