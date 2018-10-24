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

package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionWithCommit(t *testing.T) {
	gitComintStr := "4f70749fbc7b2f3576ef2c1a0e8888a4b1ef63ba"
	exp := VersionFunc + "-" + gitComintStr[:8]
	assert.Equal(t, exp, VersionWithCommit(gitComintStr))
}
