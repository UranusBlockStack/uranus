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

package main

import (
	"fmt"
	"runtime"
	rdebug "runtime/debug"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/debug"
	"github.com/spf13/cobra"
)

var memStatsCmd = &cobra.Command{
	Use:   "memstats",
	Short: "Returns detailed runtime memory statistics.",
	Long:  `Returns detailed runtime memory statistics.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(runtime.MemStats)
		cmdutils.ClientCall("Debug.MemStats", nil, &result)
		cmdutils.PrintJSON(result)
	},
}

var gcStatsCmd = &cobra.Command{
	Use:   "gcstats",
	Short: "Returns GC statistics.",
	Long:  `Returns GC statistics.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(rdebug.GCStats)
		cmdutils.ClientCall("Debug.GcStats", nil, &result)
		cmdutils.PrintJSON(result)
	},
}

var cpuProfileCmd = &cobra.Command{
	Use:   "cpuprofile <file> <nsec> ",
	Short: "Turns on CPU profiling for nsec seconds and writesprofile data to file.",
	Long:  `Turns on CPU profiling for nsec seconds and writesprofile data to file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(bool)
		var req = debug.DebugArgs{File: args[0], Nsec: cmdutils.GetUint64(args[1])}

		cmdutils.ClientCall("Debug.CpuProfile", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var goTraceCmd = &cobra.Command{
	Use:   "gotrace <file> <nsec> ",
	Short: "Turns on CPU profiling for nsec seconds and writesprofile data to file.",
	Long:  `Turns on CPU profiling for nsec seconds and writesprofile data to file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(bool)
		var req = debug.DebugArgs{File: args[0], Nsec: cmdutils.GetUint64(args[1])}

		cmdutils.ClientCall("Debug.GoTrace", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var blockProfileCmd = &cobra.Command{
	Use:   "blockprofile <file> <nsec> ",
	Short: "Turns on goroutine profiling for nsec seconds and writes profile data to file.",
	Long:  `Turns on goroutine profiling for nsec seconds and writes profile data to file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(bool)
		var req = debug.DebugArgs{File: args[0], Nsec: cmdutils.GetUint64(args[1])}

		cmdutils.ClientCall("Debug.BlockProfile", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var mutexProfileCmd = &cobra.Command{
	Use:   "mutexprofile <file> <nsec> ",
	Short: "Turns on mutex profiling for nsec seconds and writes profile data to file.",
	Long:  `Turns on mutex profiling for nsec seconds and writes profile data to file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(bool)
		var req = debug.DebugArgs{File: args[0], Nsec: cmdutils.GetUint64(args[1])}

		cmdutils.ClientCall("Debug.MutexProfile", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var writeMemProfileCmd = &cobra.Command{
	Use:   "writememprofile <file>",
	Short: "Writes an allocation profile to the given file.",
	Long:  `Writes an allocation profile to the given file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(bool)
		cmdutils.ClientCall("Debug.WriteMemProfile", args[0], &result)
		cmdutils.PrintJSON(result)
	},
}

var stacksCmd = &cobra.Command{
	Use:   "stacks",
	Short: "Returns a printed representation of the stacks of all goroutines.",
	Long:  `Returns a printed representation of the stacks of all goroutines.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var result []byte
		cmdutils.ClientCall("Debug.Stacks", nil, &result)
		fmt.Println(string(result))
	},
}

var freeOSMemoryCmd = &cobra.Command{
	Use:   "freeosmemory ",
	Short: "Returns unused memory to the OS.",
	Long:  `Returns unused memory to the OS.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var result = new(bool)
		cmdutils.ClientCall("Debug.FreeOSMemory", nil, &result)
		cmdutils.PrintJSON(result)
	},
}
