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
	"os/signal"
	"runtime"
	"syscall"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/node"
	"github.com/UranusBlockStack/uranus/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "uranus",
	Short: "uranus is a ubiquitous computing power sharing platform.",
	Long:  `uranus is a ubiquitous computing power sharing platform.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if err := unmarshalCfgFile(startConfig); err != nil {
			log.Warnf("unmarshal config file err: %v ,use default configuration.", err)
		}
		node := node.New(startConfig.NodeConfig)
		log.SetConfig(startConfig.LogConfig)
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
	falgs := RootCmd.Flags()

	// log
	falgs.StringVar(&startConfig.LogConfig.Level, "log_level", "debug", "Logging verbosity: debug, info, warn, error, fatal, panic")
	falgs.StringVar(&startConfig.LogConfig.Format, "log_format", "text", "Logging format: text,json.")

	// node
	falgs.StringVarP(&startConfig.NodeConfig.DataDir, "datadir", "d", defaultDataDir(), "Data directory for the databases")
	falgs.StringVar(&startConfig.NodeConfig.Host, "node_rpchost", startConfig.NodeConfig.Host, "HTTP and RPC server listening interface")
	falgs.IntVar(&startConfig.NodeConfig.Port, "node_rpcport", startConfig.NodeConfig.Port, "HTTP and RPC server listening port")
	falgs.StringArrayVar(&startConfig.NodeConfig.Cors, "node_rpccors", startConfig.NodeConfig.Cors, "HTTP and RPC accept cross origin requests")

	// p2p
	falgs.StringVar(&startConfig.NodeConfig.P2P.ListenAddr, "p2p_listenaddr", startConfig.NodeConfig.P2P.ListenAddr, "p2p listening port")
	falgs.IntVar(&startConfig.NodeConfig.P2P.MaxPeers, "p2p_maxpeers", startConfig.NodeConfig.P2P.MaxPeers, "maximum number of network peers")
	// falgs.StringArrayVar(&startConfig.NodeConfig.P2P.BootNodeStrs, "p2p_bootnodes", startConfig.NodeConfig.P2P.BootNodeStrs, "comma separated enode URLs for P2P discovery bootstrap")

	// config file
	falgs.StringVarP(&startConfig.CfgFile, "config", "c", "", "YAML configuration file")

	// TxPoolConfig
	falgs.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.PriceBump, "txpool_pricebump", startConfig.UranusConfig.TxPoolConfig.PriceBump, "Price bump percentage to replace an already existing transaction")
	falgs.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.PriceLimit, "txpool_pricelimit", startConfig.UranusConfig.TxPoolConfig.PriceLimit, "Minimum gas price limit to enforce for acceptance into the pool")
	falgs.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.AccountSlots, "txpool_accountslots", startConfig.UranusConfig.TxPoolConfig.AccountSlots, "Minimum number of executable transaction slots guaranteed per account")
	falgs.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.AccountQueue, "txpool_accountqueue", startConfig.UranusConfig.TxPoolConfig.AccountQueue, "Maximum number of non-executable transaction slots permitted per account")
	falgs.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.GlobalSlots, "txpool_globalslots", startConfig.UranusConfig.TxPoolConfig.GlobalSlots, "Maximum number of executable transaction slots for all accounts")
	falgs.Uint64Var(&startConfig.UranusConfig.TxPoolConfig.GlobalQueue, "txpool_globalqueue", startConfig.UranusConfig.TxPoolConfig.GlobalQueue, "Minimum number of non-executable transaction slots for all accounts")
	falgs.DurationVar(&startConfig.UranusConfig.TxPoolConfig.TimeoutDuration, "txpool_timeout", startConfig.UranusConfig.TxPoolConfig.TimeoutDuration, "Maximum amount of time non-executable transaction are queued")

	// miner
	falgs.StringVar(&startConfig.UranusConfig.MinerConfig.CoinBaseAddr, "miner_conbase", "", "Public address for block mining rewards (default = first account created)")
	falgs.StringVar(&startConfig.UranusConfig.MinerConfig.ExtraData, "miner_extradata", startConfig.UranusConfig.MinerConfig.ExtraData, "Block extra data set by the miner")
	falgs.IntVar(&startConfig.UranusConfig.MinerConfig.MinerThreads, "miner_threads", startConfig.UranusConfig.MinerConfig.MinerThreads, "Number of CPU threads to use for mining")
	falgs.BoolVar(&startConfig.UranusConfig.StartMiner, "miner_start", startConfig.UranusConfig.StartMiner, "Enable mining")

	//----------viper config file---------------

	// log

	viper.BindPFlag("log-level", falgs.Lookup("log_level"))
	viper.BindPFlag("log-format", falgs.Lookup("log_format"))

	// node
	viper.BindPFlag("node-datadir", falgs.Lookup("datadir"))
	// node.rpc
	viper.BindPFlag("rpc-host", falgs.Lookup("node_rpchost"))
	viper.BindPFlag("rpc-port", falgs.Lookup("node_rpcport"))
	viper.BindPFlag("rpc-cors", falgs.Lookup("node_rpccors"))
	// node.p2p
	viper.BindPFlag("p2p-listenaddr", falgs.Lookup("p2p_listenaddr"))
	viper.BindPFlag("p2p-maxpeers", falgs.Lookup("p2p_maxpeers"))

	// txpool
	viper.BindPFlag("txpool-pricebump", falgs.Lookup("txpool_pricebump"))
	viper.BindPFlag("txpool-pricelimit", falgs.Lookup("txpool_pricelimit"))
	viper.BindPFlag("txpool-accountslots", falgs.Lookup("txpool_accountslots"))
	viper.BindPFlag("txpool-accountqueue", falgs.Lookup("txpool_accountqueue"))
	viper.BindPFlag("txpool-globalslots", falgs.Lookup("txpool_globalslots"))
	viper.BindPFlag("txpool-globalqueue", falgs.Lookup("txpool_globalqueue"))
	viper.BindPFlag("txpool-timeout", falgs.Lookup("txpool_timeout"))

	//miner
	viper.BindPFlag("miner-conbase", falgs.Lookup("miner_conbase"))
	viper.BindPFlag("miner-extradata", falgs.Lookup("miner_extradata"))
	viper.BindPFlag("miner-threads", falgs.Lookup("miner_threads"))
	viper.BindPFlag("miner-start", falgs.Lookup("miner_start"))
}

func initConfig() {
	if startConfig.CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(startConfig.CfgFile)
	} else {
		log.Info("No config file , use default configuration.")
		return
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Infof("Can't read config: %v, use default configuration.", err)
		return
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
