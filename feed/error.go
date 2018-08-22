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

package feed

import (
	"errors"
	"reflect"
)

var errBadChannel = errors.New("event: Subscribe argument does not have sendable channel type")

type feedTypeError struct {
	got, want reflect.Type
	op        string
}

func newFeedTypeError(op string, got, want reflect.Type) feedTypeError {
	return feedTypeError{
		op:   op,
		got:  got,
		want: want,
	}
}

func (e feedTypeError) Error() string {
	return "event: wrong type in " + e.op + " got " + e.got.String() + ", want " + e.want.String()
}

func (f *Feed) init() {
	f.sendLock = make(chan struct{}, 1)
	f.removeSub = make(chan interface{})
	f.sendLock <- struct{}{}
	f.sendCases = cases{{Chan: reflect.ValueOf(f.removeSub), Dir: reflect.SelectRecv}}
}
