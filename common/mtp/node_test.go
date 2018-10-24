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

package mtp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanUnload(t *testing.T) {
	tests := []struct {
		flag       nodeFlag
		gen, limit uint16
		expected   bool
	}{
		{
			flag: nodeFlag{dirty: false, gen: 65534},
			gen:  65535, limit: 1,
			expected: true,
		},
		{
			flag: nodeFlag{dirty: false, gen: 0},
			gen:  0, limit: 0,
			expected: true,
		},
		{
			flag: nodeFlag{dirty: false, gen: 65534},
			gen:  0, limit: 1,
			expected: true,
		},
		{
			flag: nodeFlag{dirty: false, gen: 1},
			gen:  65535, limit: 1,
			expected: true,
		},
		{
			flag:     nodeFlag{dirty: true, gen: 0},
			expected: false,
		},
	}

	for _, test := range tests {
		got := test.flag.canUnload(test.gen, test.limit)
		assert.Equal(t, test.expected, got)
	}
}
