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
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	//Identifier  identifier to advertise over the network
	Identifier = "uranus"
)

const (
	// VersionMajor is Major version component of the current release
	VersionMajor = 0
	// VersionMinor is Minor version component of the current release
	VersionMinor = 1
	// VersionPatch is Patch version component of the current release
	VersionPatch = 0
)

// VersionFunc holds the textual version string.
var VersionFunc = func() string { return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch) }()

// GitCommit  Git SHA1 commit hash of the release
var GitCommit = func() string {
	head := readGit("HEAD")
	if splits := strings.Split(head, " "); len(splits) == 2 {
		head = splits[1]
		return readGit(head)
	}
	return ""
}

// readGit returns content of file in .git directory.
func readGit(file string) string {
	content, err := ioutil.ReadFile(path.Join(".git", file))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

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
