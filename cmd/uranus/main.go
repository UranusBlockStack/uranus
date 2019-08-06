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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/debug"
	"github.com/UranusBlockStack/uranus/node"
	"github.com/UranusBlockStack/uranus/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	errNoConfigFile    error
	errViperReadConfig error
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "uranus",
	Short: "uranus is a ubiquitous computing power sharing platform.",
	Long:  `uranus is a ubiquitous computing power sharing platform.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		err := unmarshalCfgFile(startConfig)
		log.SetConfig(startConfig.LogConfig)

		if errNoConfigFile != nil {
			log.Info(errNoConfigFile)
		}

		if errViperReadConfig != nil {
			log.Warn(errViperReadConfig)
		}

		if err != nil {
			log.Warnf("unmarshal config file err: %v ,use default configuration.", err)
		}

		if err := debug.Setup(startConfig.DebugConfig); err != nil {
			log.Errorf("uranus start debug model failed err: %v", err)
		}

		node := node.New(startConfig.NodeConfig)
		if err := registerService(node); err != nil {
			log.Errorf("uranus register service failed err: %v", err)
		}
		if err := startNode(node); err != nil {
			log.Errorf("uranus start node failed err: %v", err)
		}
		node.Wait()
	},
}

// start up the node itself
func startNode(stack *node.Node) error {
	log.Info("uranus start...")
	if err := stack.Start(); err != nil {
		return err
	}
	go stopNode(stack)
	return nil
}

func stopNode(stack *node.Node) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)
	<-sigc
	log.Info("Got interrupt, shutting down...")
	go stack.Stop()
	for i := 3; i > 0; i-- {
		<-sigc
		if i > 1 {
			log.Warnf("Already shutting down, interrupt more to panic times: %v", i-1)
		}
	}
}

func registerService(stack *node.Node) error {
	var err error
	// register uranus server
	err = stack.Register(func(ctx *node.Context) (node.Service, error) {
		return server.New(ctx, startConfig.UranusConfig)
	})
	return err
}

func init() {
	cobra.OnInitialize(initConfig)
	flags := RootCmd.Flags()

	// log
	flags.StringVar(&startConfig.LogConfig.Level, "log_level", "info", "Logging verbosity: debug, info, warn, error, fatal, panic")
	flags.StringVar(&startConfig.LogConfig.Format, "log_format", "text", "Logging format: text,json.")

	// node
	flags.StringVarP(&startConfig.NodeConfig.DataDir, "datadir", "d", cmdutils.DefaultDataDir(), "Data directory for the databases")
	flags.StringVar(&startConfig.NodeConfig.Host, "node_rpchost", startConfig.NodeConfig.Host, "HTTP and RPC server listening interface")
	flags.IntVar(&startConfig.NodeConfig.Port, "node_rpcport", startConfig.NodeConfig.Port, "HTTP and RPC server listening port")
	flags.StringSliceVar(&startConfig.NodeConfig.Cors, "node_rpccors", startConfig.NodeConfig.Cors, "HTTP and RPC accept cross origin requests")
	flags.StringVar(&startConfig.NodeConfig.WSHost, "node_wshost", startConfig.NodeConfig.WSHost, "Websocket server listening interface")
	flags.IntVar(&startConfig.NodeConfig.WSPort, "node_wsport", startConfig.NodeConfig.WSPort, "Websocket server listening port")
	flags.StringSliceVar(&startConfig.NodeConfig.WSOrigins, "node_wsorigins", startConfig.NodeConfig.WSOrigins, "Websocket accept cross origin requests")

	// p2p
	flags.StringVar(&startConfig.NodeConfig.P2P.ListenAddr, "p2p_listenaddr", startConfig.NodeConfig.P2P.ListenAddr, "p2p listening url")
	flags.IntVar(&startConfig.NodeConfig.P2P.MaxPeers, "p2p_maxpeers", startConfig.NodeConfig.P2P.MaxPeers, "maximum number of network peers")
	flags.StringSliceVar(&startConfig.NodeConfig.P2P.BootNodeStrs, "p2p_bootnodes", startConfig.NodeConfig.P2P.BootNodeStrs, "comma separated enode URLs for P2P discovery bootstrap")

	// config file
	flags.StringVarP(&startConfig.CfgFile, "config", "c", "", "YAML configuration file")
	flags.StringVarP(&startConfig.GenesisFile, "genesis", "g", "", "YAML configuration file")

	// TxPoolConfig
	flags.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.PriceBump, "txpool_pricebump", startConfig.UranusConfig.TxPoolConfig.PriceBump, "Price bump percentage to replace an already existing transaction")
	flags.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.PriceLimit, "txpool_pricelimit", startConfig.UranusConfig.TxPoolConfig.PriceLimit, "Minimum gas price limit to enforce for acceptance into the pool")
	flags.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.AccountSlots, "txpool_accountslots", startConfig.UranusConfig.TxPoolConfig.AccountSlots, "Minimum number of executable transaction slots guaranteed per account")
	flags.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.AccountQueue, "txpool_accountqueue", startConfig.UranusConfig.TxPoolConfig.AccountQueue, "Maximum number of non-executable transaction slots permitted per account")
	flags.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.GlobalSlots, "txpool_globalslots", startConfig.UranusConfig.TxPoolConfig.GlobalSlots, "Maximum number of executable transaction slots for all accounts")
	flags.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.GlobalQueue, "txpool_globalqueue", startConfig.UranusConfig.TxPoolConfig.GlobalQueue, "Minimum number of non-executable transaction slots for all accounts")
	flags.DurationVar(&startConfig.UranusConfig.TxPoolConfig.TimeoutDuration, "txpool_timeout", startConfig.UranusConfig.TxPoolConfig.TimeoutDuration, "Maximum amount of time non-executable transaction are queued")

	// miner
	flags.StringVar(&startConfig.UranusConfig.MinerConfig.CoinBaseAddr, "miner_conbase", "", "Public address for block mining rewards (default = first account created)")
	flags.StringVar(&startConfig.UranusConfig.MinerConfig.ExtraData, "miner_extradata", startConfig.UranusConfig.MinerConfig.ExtraData, "Block extra data set by the miner")
	flags.IntVar(&startConfig.UranusConfig.MinerConfig.MinerThreads, "miner_threads", startConfig.UranusConfig.MinerConfig.MinerThreads, "Number of CPU threads to use for mining")
	flags.BoolVar(&startConfig.UranusConfig.StartMiner, "miner_start", startConfig.UranusConfig.StartMiner, "Enable mining")

	// debug
	flags.BoolVar(&startConfig.DebugConfig.Pprof, "debug_pprof", startConfig.DebugConfig.Pprof, "Enable the pprof HTTP server")
	flags.IntVar(&startConfig.DebugConfig.PprofPort, "debug_pprof_port", startConfig.DebugConfig.PprofPort, "Pprof HTTP server listening port")
	flags.StringVar(&startConfig.DebugConfig.PprofAddr, "debug_pprof_addr", startConfig.DebugConfig.PprofAddr, "Pprof HTTP server listening interface")
	flags.IntVar(&startConfig.DebugConfig.Memprofilerate, "debug_memprofilerate", startConfig.DebugConfig.Memprofilerate, "Turn on memory profiling with the given rate")
	flags.IntVar(&startConfig.DebugConfig.Blockprofilerate, "debug_blockprofilerate", startConfig.DebugConfig.Blockprofilerate, "Turn on block profiling with the given rate")
	flags.StringVar(&startConfig.DebugConfig.Cpuprofile, "debug_cpuprofile", startConfig.DebugConfig.Cpuprofile, "Write CPU profile to the given file")
	flags.StringVar(&startConfig.DebugConfig.Trace, "debug_trace", startConfig.DebugConfig.Trace, "Write execution trace to the given file")

	//----------viper config file---------------

	// log

	viper.BindPFlag("log-level", flags.Lookup("log_level"))
	viper.BindPFlag("log-format", flags.Lookup("log_format"))

	// node
	viper.BindPFlag("node-datadir", flags.Lookup("datadir"))
	// node.rpc
	viper.BindPFlag("rpc-host", flags.Lookup("node_rpchost"))
	viper.BindPFlag("rpc-port", flags.Lookup("node_rpcport"))
	viper.BindPFlag("rpc-cors", flags.Lookup("node_rpccors"))
	// node.websocket
	viper.BindPFlag("ws-host", flags.Lookup("node_wshost"))
	viper.BindPFlag("ws-port", flags.Lookup("node_wsport"))
	viper.BindPFlag("ws-origins", flags.Lookup("node_wsorigins"))
	// node.p2p
	viper.BindPFlag("p2p-listenaddr", flags.Lookup("p2p_listenaddr"))
	viper.BindPFlag("p2p-maxpeers", flags.Lookup("p2p_maxpeers"))

	// txpool
	viper.BindPFlag("txpool-pricebump", flags.Lookup("txpool_pricebump"))
	viper.BindPFlag("txpool-pricelimit", flags.Lookup("txpool_pricelimit"))
	viper.BindPFlag("txpool-accountslots", flags.Lookup("txpool_accountslots"))
	viper.BindPFlag("txpool-accountqueue", flags.Lookup("txpool_accountqueue"))
	viper.BindPFlag("txpool-globalslots", flags.Lookup("txpool_globalslots"))
	viper.BindPFlag("txpool-globalqueue", flags.Lookup("txpool_globalqueue"))
	viper.BindPFlag("txpool-timeout", flags.Lookup("txpool_timeout"))

	// miner
	viper.BindPFlag("miner-conbase", flags.Lookup("miner_conbase"))
	viper.BindPFlag("miner-extradata", flags.Lookup("miner_extradata"))
	viper.BindPFlag("miner-threads", flags.Lookup("miner_threads"))
	viper.BindPFlag("miner-start", flags.Lookup("miner_start"))

	// debug
	viper.BindPFlag("debug-pprof", flags.Lookup("debug_pprof"))
	viper.BindPFlag("debug-pprofport", flags.Lookup("debug_pprof_port"))
	viper.BindPFlag("debug-pprofaddr", flags.Lookup("debug_pprof_addr"))
	viper.BindPFlag("debug-memprofilerate", flags.Lookup("debug_memprofilerate"))
	viper.BindPFlag("debug-blockprofilerate", flags.Lookup("debug_blockprofilerate"))
	viper.BindPFlag("debug-cpuprofile", flags.Lookup("debug_cpuprofile"))
	viper.BindPFlag("debug-trace", flags.Lookup("debug_trace"))
}

func initConfig() {
	if startConfig.CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(startConfig.CfgFile)
	} else {
		errNoConfigFile = errors.New("No config file , use default configuration")
		return
	}

	if err := viper.ReadInConfig(); err != nil {
		errViperReadConfig = fmt.Errorf("Can't read config: %v, use default configuration", err)
	}
}

func unmarshalCfgFile(startConfig *StartConfig) error {

	// log
	if err := viper.Unmarshal(startConfig.LogConfig); err != nil {
		return err
	}

	// node
	if err := viper.Unmarshal(startConfig.NodeConfig); err != nil {
		return err
	}

	// p2p
	if err := viper.Unmarshal(startConfig.NodeConfig.P2P); err != nil {
		return err
	}

	// uranus
	if err := viper.Unmarshal(startConfig.UranusConfig); err != nil {
		return err
	}

	// txpool
	if err := viper.Unmarshal(startConfig.UranusConfig.TxPoolConfig); err != nil {
		return err
	}

	// miner
	if err := viper.Unmarshal(startConfig.UranusConfig.MinerConfig); err != nil {
		return err
	}

	// debug
	if err := viper.Unmarshal(startConfig.DebugConfig); err != nil {
		return err
	}

	// Make sure we have a valid genesis JSON
	if len(startConfig.GenesisFile) != 0 {
		file, err := os.Open(startConfig.GenesisFile)
		if err != nil {
			return fmt.Errorf("Failed to read genesis file: %v(%v)", startConfig.GenesisFile, err)
		}
		defer file.Close()

		genesis := new(ledger.Genesis)
		if err := json.NewDecoder(file).Decode(genesis); err != nil {
			return fmt.Errorf("invalid genesis file: %v(%v)", startConfig.GenesisFile, err)
		}
		startConfig.UranusConfig.Genesis = genesis
	}
	return nil
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	Execute()
}
