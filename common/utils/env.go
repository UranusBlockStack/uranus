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

package utils

import "os"

var funcs []func() bool

func EnvParse() {
	ok := true
	for _, f := range funcs {
		ok = f() && ok
	}
	if !ok {
		os.Exit(1)
	}
}

func EnvString(name string, value string) *string {
	p := new(string)
	EnvStringVar(p, name, value)
	return p
}

func EnvStringVar(p *string, name string, value string) {
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			*p = s
		}
		return true
	})
}
