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

package node

import (
	"fmt"
	"path/filepath"

	"github.com/UranusBlockStack/uranus/p2p"
)

var DefaultName = "uranus"

// Config config of node
type Config struct {
	Name    string
	DataDir string `mapstructure:"node-datadir"`

	Host string   `mapstructure:"rpc-host"`
	Port int      `mapstructure:"rpc-port"`
	Cors []string `mapstructure:"rpc-cors"`

	WSHost    string   `mapstructure:"ws-host"`
	WSPort    int      `mapstructure:"ws-port"`
	WSOrigins []string `mapstructure:"ws-origins"`

	P2P *p2p.Config
}

// NewConfig initialize node config
func NewConfig(dataDir string) *Config {
	return &Config{
		Name:    DefaultName,
		DataDir: dataDir,
		P2P:     &p2p.Config{},
	}
}

//Endpoint resolves an endpoint based on the configured host interface and port parameters.
func (c *Config) Endpoint() string {
	if c.Host == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// WSEndpoint resolves a websocket endpoint based on the configured host interface
// and port parameters.
func (c *Config) WSEndpoint() string {
	if c.WSHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.WSHost, c.WSPort)
}

// resolvePath resolves path in the instance directory.
func (c *Config) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Join(c.DataDir, c.Name), path)
}
