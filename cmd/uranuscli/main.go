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
	"os"
	"runtime"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "uranuscli",
	Short: "uranuscli is a commond line client for uranus.",
	Long:  `uranuscli is a commond line client for uranus.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	utils.EnvParse()

	RootCmd.PersistentFlags().StringVarP(&cmdutils.CoreURL, "curl", "c", *cmdutils.DefaultCoreURL, "uranus server URL.")
	RootCmd.PersistentFlags().BoolVarP(&cmdutils.OneLine, "oneline", "o", false, "Streamline pattern, Output less and better content.")
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	AddCommands()
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	Execute()
}
