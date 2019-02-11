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
	"io"
	"os"

	"github.com/Sirupsen/logrus"
)

// Config log config
type Config struct {
	Format    string `mapstructure:"log-format"`
	Formatter logrus.Formatter
	Level     string `mapstructure:"log-level"`
	Output    io.Writer
}

// DefaultConfig default config
var DefaultConfig = &Config{
	Format: "text",
	Formatter: &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.999",
		FullTimestamp:   true,
	},
	Level:  "debug",
	Output: os.Stderr,
}
