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

package txpool

import "time"

// Config are the configuration parameters of the transaction pool.
type Config struct {
	PriceLimit uint64 `mapstructure:"txpool-pricelimit"`
	PriceBump  uint64 `mapstructure:"txpool-pricebump"`

	AccountSlots uint64 `mapstructure:"txpool-accountslots"`
	GlobalSlots  uint64 `mapstructure:"txpool-globalslots"`
	AccountQueue uint64 `mapstructure:"txpool-accountqueue"`
	GlobalQueue  uint64 `mapstructure:"txpool-globalqueue"`

	TimeoutDuration time.Duration `mapstructure:"txpool-timeout"`
}

// DefaultTxPoolConfig contains the default configurations for the transaction
// pool.
var DefaultTxPoolConfig = Config{
	PriceLimit: 1,
	PriceBump:  10,

	AccountSlots: 16,
	GlobalSlots:  4096,
	AccountQueue: 64,
	GlobalQueue:  1024,

	TimeoutDuration: 3 * time.Hour,
}
