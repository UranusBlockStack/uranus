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
	"bytes"
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
)

// Handler is the global debugging handler.
var Handler = new(HandlerT)

// HandlerT implements the debugging API.
// Do not create values of this type, use the one
// in the Handler variable instead.
type HandlerT struct {
	mu        sync.Mutex
	cpuW      io.WriteCloser
	cpuFile   string
	traceW    io.WriteCloser
	traceFile string
}

// MemStats returns detailed runtime memory statistics.
func (*HandlerT) MemStats(ignore string, reply *runtime.MemStats) error {
	runtime.ReadMemStats(reply)
	return nil
}

// GcStats returns GC statistics.
func (*HandlerT) GcStats(ignore string, reply *debug.GCStats) error {
	debug.ReadGCStats(reply)
	return nil
}

type DebugArgs struct {
	File string
	Nsec uint64
}

// CpuProfile turns on CPU profiling for nsec seconds and writesprofile data to file.
func (h *HandlerT) CpuProfile(args DebugArgs, reply *bool) error {
	if err := h.StartCPUProfile(args.File, reply); err != nil {
		return err
	}
	time.Sleep(time.Duration(args.Nsec) * time.Second)

	err := h.StopCPUProfile("", reply)
	*reply = err == nil
	return err
}

// StartCPUProfile turns on CPU profiling, writing to the given file.
func (h *HandlerT) StartCPUProfile(file string, reply *bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cpuW != nil {
		return errors.New("CPU profiling already in progress")
	}
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return err
	}
	h.cpuW = f
	h.cpuFile = file
	log.Infof("CPU profiling started dump: %v", h.cpuFile)
	*reply = true
	return nil
}

// StopCPUProfile stops an ongoing CPU profile.
func (h *HandlerT) StopCPUProfile(ignore string, reply *bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	pprof.StopCPUProfile()
	if h.cpuW == nil {
		return errors.New("CPU profiling not in progress")
	}
	log.Infof("Done writing CPU profile dump: %v", h.cpuFile)
	h.cpuW.Close()
	h.cpuW = nil
	h.cpuFile = ""
	*reply = true
	return nil
}

// GoTrace turns on tracing for nsec seconds and writes
// trace data to file.
func (h *HandlerT) GoTrace(args DebugArgs, reply *bool) error {
	if err := h.StartGoTrace(args.File, reply); err != nil {
		return err
	}
	time.Sleep(time.Duration(args.Nsec) * time.Second)
	err := h.StopGoTrace("", reply)
	*reply = err == nil
	return err
}

// StartGoTrace turns on tracing, writing to the given file.
func (h *HandlerT) StartGoTrace(file string, reply *bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.traceW != nil {
		return errors.New("trace already in progress")
	}
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	if err := trace.Start(f); err != nil {
		f.Close()
		return err
	}
	h.traceW = f
	h.traceFile = file
	log.Infof("Go tracing started dump: %v", h.traceFile)
	*reply = true
	return nil
}

// StopGoTrace stops an ongoing trace.
func (h *HandlerT) StopGoTrace(ignore string, reply *bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	trace.Stop()
	if h.traceW == nil {
		return errors.New("trace not in progress")
	}
	log.Infof("Done writing Go trace dump: %v", h.traceFile)
	h.traceW.Close()
	h.traceW = nil
	h.traceFile = ""
	*reply = true
	return nil
}

// BlockProfile turns on goroutine profiling for nsec seconds and writes profile data to
// file. It uses a profile rate of 1 for most accurate information. If a different rate is
// desired, set the rate and write the profile manually.
func (*HandlerT) BlockProfile(args DebugArgs, reply *bool) error {
	runtime.SetBlockProfileRate(1)
	time.Sleep(time.Duration(args.Nsec) * time.Second)
	defer runtime.SetBlockProfileRate(0)
	err := writeProfile("block", args.File)
	*reply = err == nil
	return err
}

// MutexProfile turns on mutex profiling for nsec seconds and writes profile data to file.
// It uses a profile rate of 1 for most accurate information. If a different rate is
// desired, set the rate and write the profile manually.
func (*HandlerT) MutexProfile(args DebugArgs, reply *bool) error {
	runtime.SetMutexProfileFraction(1)
	time.Sleep(time.Duration(args.Nsec) * time.Second)
	defer runtime.SetMutexProfileFraction(0)
	err := writeProfile("mutex", args.File)
	*reply = err == nil
	return err
}

// WriteMemProfile writes an allocation profile to the given file.
// Note that the profiling rate cannot be set through the API,
// it must be set on the command line.
func (*HandlerT) WriteMemProfile(file string, reply *bool) error {
	err := writeProfile("heap", file)
	*reply = err == nil
	return err
}

// Stacks returns a printed representation of the stacks of all goroutines.
func (*HandlerT) Stacks(ignore string, reply *[]byte) error {
	buf := new(bytes.Buffer)
	pprof.Lookup("goroutine").WriteTo(buf, 2)
	*reply = buf.Bytes()
	return nil
}

// FreeOSMemory returns unused memory to the OS.
func (*HandlerT) FreeOSMemory(ignore string, reply *bool) error {
	debug.FreeOSMemory()
	*reply = true
	return nil
}

// SetGCPercent sets the garbage collection target percentage. It returns the previous
// setting. A negative value disables GC.
func (*HandlerT) SetGCPercent(v int, reply *int) error {
	*reply = debug.SetGCPercent(v)
	return nil
}

// setBlockProfileRate sets the rate of goroutine block profile data collection.
// rate 0 disables block profiling.
func (*HandlerT) setBlockProfileRate(rate int) {
	runtime.SetBlockProfileRate(rate)
}

func writeProfile(name, file string) error {
	p := pprof.Lookup(name)
	log.Infof("Writing profile records count: %v, type: %v, dump: %v", p.Count(), name, file)
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	defer f.Close()
	return p.WriteTo(f, 0)
}

// expands home directory in file paths.
// ~someuser/tmp will not be expanded.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home := os.Getenv("HOME")
		if home == "" {
			if usr, err := user.Current(); err == nil {
				home = usr.HomeDir
			}
		}
		if home != "" {
			p = home + p[1:]
		}
	}
	return filepath.Clean(p)
}
