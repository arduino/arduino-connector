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
	var v []DylibMap

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
	bytes, err := json.MarshalIndent(v, "", "   ")
	if err == nil {
		fmt.Println(string(bytes))
	}
}
