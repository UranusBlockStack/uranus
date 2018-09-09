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

package log

import (
	"fmt"
	"reflect"
)

func formatArgs(args []interface{}) []interface{} {
	results := make([]interface{}, len(args))
	f := func(value interface{}) (result interface{}) {
		defer func() {
			if err := recover(); err != nil {
				if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
					result = "nil"
				} else {
					panic(err)
				}
			}
		}()
		switch v := value.(type) {
		case error:
			return v.Error()

		case fmt.Stringer:
			return v.String()
		default:
			return v
		}
	}

	for k, v := range args {
		results[k] = f(v)
	}
	return results
}
