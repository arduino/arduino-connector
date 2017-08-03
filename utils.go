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
			if strings.Contains(f.Name(), ".so") {
				extraPaths = appendIfUnique(extraPaths, filepath.Dir(path))
			}
			return nil
		})
		os.Setenv("LD_LIBRARY_PATH", strings.Join(extraPaths, ":")+":"+os.Getenv("LD_LIBRARY_PATH"))
	}
}
