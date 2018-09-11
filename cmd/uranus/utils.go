// Copyright 2018 The uranus Authors
// This file is part of the uranus library.
//
// The uranus library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The uranus library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the uranus library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func defaultDataDir() string {
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "uranus_dir")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "uranus_dir")
		} else {
			return filepath.Join(home, ".uranus_dir")
		}
	}
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
