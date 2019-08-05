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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/filelock"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/UranusBlockStack/uranus/p2p/discover"
	"github.com/UranusBlockStack/uranus/params"
	"github.com/UranusBlockStack/uranus/rpc"
)

type communication struct {
	endpoint string
	listener net.Listener
	handler  *rpc.Server
	cors     []string
	origins  []string
}

type Constructor func(ctx *Context) (Service, error)

// Node is a container on which services can be registered.
type Node struct {
	config    *Config
	p2pConfig *p2p.Config

	rpc, websocket *communication

	running         bool
	instanceDirLock filelock.Releaser
	serviceFuncs    []Constructor            // Service constructors
	services        map[reflect.Type]Service // Currently running services
	stop            chan struct{}            // Channel to stop
	lock            sync.RWMutex
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) *Node {
	return &Node{
		config:       conf,
		p2pConfig:    conf.P2P,
		rpc:          &communication{endpoint: conf.Endpoint(), cors: conf.Cors},
		websocket:    &communication{endpoint: conf.WSEndpoint(), origins: conf.WSOrigins},
		running:      false,
		serviceFuncs: []Constructor{},
		services:     make(map[reflect.Type]Service),
	}
}

// Register injects a new service into the node's stack.
func (n *Node) Register(constructor Constructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.running {
		return ErrNodeRunning
	}
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

// Start create a live node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.running {
		return ErrNodeRunning
	}

	if err := n.openDataDir(); err != nil {
		return err
	}

	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &Context{
			config:   n.config,
			services: make(map[reflect.Type]Service),
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return fmt.Errorf("duplicate service: %v", kind)
		}
		services[kind] = service
	}

	// create p2p server
	p2pServer := &p2p.Server{Config: *n.p2pConfig}
	if len(p2pServer.BootNodeStrs) == 0 {
		for _, url := range params.BootNodes {
			node, err := discover.ParseNode(url)
			if err != nil {
				log.Error("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			p2pServer.BootNodes = append(p2pServer.BootNodes, node)
		}
	} else {
		for _, url := range p2pServer.BootNodeStrs {
			node, err := discover.ParseNode(url)
			if err != nil {
				log.Error("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			p2pServer.BootNodes = append(p2pServer.BootNodes, node)
		}
	}
	if p2pServer.PrivateKey == nil {
		if file := "nodekey"; utils.FileExists(file) {
			if key, err := crypto.LoadECDSA(file); err != nil {
				log.Fatalf("failed to open %s: %v", file, err)
			} else {
				p2pServer.PrivateKey = key
			}
		} else if file = filepath.Join(n.config.DataDir, "nodekey"); utils.FileExists(file) {
			if key, err := crypto.LoadECDSA(file); err != nil {
				log.Fatalf("failed to open %s: %v", file, err)
			} else {
				p2pServer.PrivateKey = key
			}
		} else {
			nodeKey, err := crypto.GenerateKey()
			if err != nil {
				log.Fatalf("could not generate key: %v", err)
			}
			if err = crypto.SaveECDSA(file, nodeKey); err != nil {
				log.Fatalf("%v", err)
			}
			p2pServer.PrivateKey = nodeKey
		}
	}
	if p2pServer.NodeDatabase == "" {
		p2pServer.NodeDatabase = n.config.resolvePath("nodes")
	}
	for _, service := range services {
		p2pServer.Protocols = append(p2pServer.Protocols, service.Protocols()...)
	}

	if err := p2pServer.Start(); err != nil {
		log.Errorf("p2p start failed --- %v", err)
		return err
	}

	// Start each of the services
	started := []reflect.Type{}
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(p2pServer); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}

	// start rpc
	var apis []rpc.API
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	if err := n.startRPC(apis); err != nil {
		n.stopRPC()
		return err
	}

	if err := n.startWS(apis); err != nil {
		n.stopWS()
		return err
	}

	n.services = services
	n.running = true
	n.stop = make(chan struct{})
	return nil
}

// startRPC initializes and starts the  RPC endpoint.
func (n *Node) startRPC(apis []rpc.API) error {
	if n.rpc.endpoint == "" {
		return nil // RPC disabled.
	}
	listener, _, err := rpc.StartRPCAndHTTP(n.rpc.endpoint, apis, n.rpc.cors)
	if err != nil {
		return err
	}
	n.rpc.listener = listener

	log.Infof("RPC and HTTP endpoint opened: %v", n.rpc.endpoint)
	return nil
}

func (n *Node) stopRPC() {
	if n.rpc.listener != nil {
		n.rpc.listener.Close()
		n.rpc.listener = nil
	}
}

// startWS initializes and starts the  websocket endpoint.
func (n *Node) startWS(apis []rpc.API) error {
	if n.websocket.endpoint == "" {
		return nil // RPC disabled.
	}
	listener, _, err := rpc.StartWSEndpoint(n.websocket.endpoint, n.websocket.origins, apis)
	if err != nil {
		return err
	}
	n.websocket.listener = listener

	log.Infof("Websocket endpoint opened: %v", n.websocket.endpoint)
	return nil
}

func (n *Node) stopWS() {
	if n.websocket.listener != nil {
		n.websocket.listener.Close()
		n.websocket.listener = nil
	}
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's not running
	if !n.running {
		return ErrNodeStopped
	}

	n.stopWS()
	n.stopRPC()

	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.services = nil

	n.releaseInstanceDir()

	close(n.stop)

	n.running = false

	if len(failure.Services) > 0 {
		return failure
	}

	return nil
}

// Wait blocks the thread until the node is stopped.
func (n *Node) Wait() {
	n.lock.RLock()
	if !n.running {
		n.lock.RUnlock()
		return
	}
	stop := n.stop
	n.lock.RUnlock()
	<-stop
}

// Restart terminates a running node .
func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	return n.Start()
}

// Service retrieves a currently running service registered of a specific type.
func (n *Node) Service(service interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Short circuit if the node's not running
	if !n.running {
		return ErrNodeStopped
	}
	// Otherwise try to find the service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := n.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

func (n *Node) openDataDir() error {
	if n.config.DataDir == "" {
		return nil
	}
	instdir := filepath.Join(n.config.DataDir, n.config.Name)
	if err := os.MkdirAll(instdir, 0700); err != nil {
		return err
	}
	return n.lockInstanceDir(instdir)
}

// lockInstanceDir Lock the instance directory.
func (n *Node) lockInstanceDir(path string) error {
	release, _, err := filelock.New(filepath.Join(path, "LOCK"))
	if err != nil {
		return filelock.CheckError(err)
	}
	n.instanceDirLock = release
	return nil
}

// releaseInstanceDir Release instance directory lock.
func (n *Node) releaseInstanceDir() {
	if n.instanceDirLock != nil {
		if err := n.instanceDirLock.Release(); err != nil {
			log.Error("Can't release datadir lock", "err", err)
		}
		n.instanceDirLock = nil
	}
}
