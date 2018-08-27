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

package forecast

import "math/big"

type Config struct {
	BlockNum int // the number of block
	Percent  int
	GasPrice *big.Int
}

var DefaultConfig = &Config{
	BlockNum: 20,
	Percent:  260,
	GasPrice: big.NewInt(18 * 1e9),
}

func (c *Config) check() *Config {
	// check BlockNum
	if c.BlockNum < 1 {
		c.BlockNum = 1
	}
	// check Percent
	if c.Percent < 0 {
		c.Percent = 0
	}
	if c.Percent > 100 {
		c.Percent = 100
	}
	return c
}
