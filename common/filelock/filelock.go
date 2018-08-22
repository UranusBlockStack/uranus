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

// Package filelock provides portable file locking.
package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

var (
	datadirInUseErrnos = map[uint]bool{11: true, 32: true, 35: true}
	// ErrDatadirUsed datadir already used by another process,datadir can only be accessed by a process.
	ErrDatadirUsed = errors.New("datadir already used by another process")
)

// CheckError check file lock err
func CheckError(err error) error {
	if errno, ok := err.(syscall.Errno); ok && datadirInUseErrnos[uint(errno)] {
		return ErrDatadirUsed
	}
	return err
}

// Releaser provides the Release method to release a file lock.
type Releaser interface {
	Release() error
}

// New locks the file with the provided name.
func New(fileName string) (r Releaser, existed bool, err error) {
	if err = os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return
	}

	_, err = os.Stat(fileName)
	existed = err == nil

	r, err = newLock(fileName)
	return
}
