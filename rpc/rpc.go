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

package rpc

import (
	"net"

	"github.com/UranusBlockStack/uranus/common/log"
)

// StartRPCAndHTTP start RPC and HTTP service
func StartRPCAndHTTP(endpoint string, apis []API, cors []string) (net.Listener, *Server, error) {
	var (
		listener net.Listener
		err      error
		server   = NewServer()
	)
	log.Infof("rpc and http listen endpoint: %v", endpoint)
	for _, api := range apis {
		if err := server.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, nil, err
		}
	}
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, err
	}
	go NewHTTPServer(server, cors).Serve(listener)

	return listener, server, nil
}
