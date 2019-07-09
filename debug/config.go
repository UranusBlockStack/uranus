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

package debug

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/fjl/memsize/memsizeui"
)

var Memsize memsizeui.Handler

type Config struct {
	Pprof            bool   `mapstructure:"debug-pprof"`
	PprofPort        int    `mapstructure:"debug-pprofport"`
	PprofAddr        string `mapstructure:"debug-pprofaddr"`
	Memprofilerate   int    `mapstructure:"debug-memprofilerate"`
	Blockprofilerate int    `mapstructure:"debug-blockprofilerate"`
	Cpuprofile       string `mapstructure:"debug-cpuprofile"`
	Trace            string `mapstructure:"debug-trace"`
}

func DefaultConfig() *Config {
	return &Config{
		Pprof:          false,
		PprofPort:      6060,
		PprofAddr:      "localhost",
		Memprofilerate: runtime.MemProfileRate,
	}
}

// Setup initializes profiling ,It should be called
// as early as possible in the program.
func Setup(debugCfg *Config) error {
	// profiling, tracing
	runtime.MemProfileRate = debugCfg.Memprofilerate
	Handler.setBlockProfileRate(debugCfg.Blockprofilerate)
	if debugCfg.Trace != "" {
		if err := Handler.StartGoTrace(debugCfg.Trace, new(bool)); err != nil {
			return err
		}
	}
	if debugCfg.Cpuprofile != "" {
		if err := Handler.StartCPUProfile(debugCfg.Cpuprofile, new(bool)); err != nil {
			return err
		}
	}

	// pprof server
	if debugCfg.Pprof {
		address := fmt.Sprintf("%s:%d", debugCfg.PprofAddr, debugCfg.PprofPort)
		StartPProf(address)
	}
	return nil
}

func StartPProf(address string) {
	http.Handle("/memsize/", http.StripPrefix("/memsize", &Memsize))
	log.Infof("Starting pprof server addr: %v", fmt.Sprintf("http://%s/debug/pprof", address))
	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Errorf("Failure in running pprof server err: %v", err)
		}
	}()
}

// Exit stops all running profiles, flushing their output to the
// respective file.
func Exit() {
	Handler.StopCPUProfile("", new(bool))
	Handler.StopGoTrace("", new(bool))
}
