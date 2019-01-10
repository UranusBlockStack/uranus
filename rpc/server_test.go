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
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

type Service1Request struct {
	A int
	B int
}

type Service1Response struct {
	Result int
}

type Service1 struct {
}

func (t *Service1) Multiply(r *http.Request, req *Service1Request, res *Service1Response) error {
	res.Result = req.A * req.B
	return nil
}

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}
type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B * 100
	return nil
}
func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

type Argst struct {
	A, B int
}

func TestServer(t *testing.T) {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", ":8000"); err != nil {
		t.Errorf("TestServer error %+v", err)
	}
	server := NewServer()
	server.Register(new(Arith))
	go NewHTTPServer(server, []string{"*"}).Serve(listener)
	// curl -X POST  -d '{"id": 1, "method": "Arith.Multiply", "params":[{"A":1, "B":3}]}' http://localhost:8000/

	client, err := DialHTTP("http://127.0.0.1:8000")
	var result int
	fmt.Println("client.Call", client.Call("Arith.Multiply", Argst{A: 1, B: 2}, &result))
	fmt.Println("result", result, err)
	time.Sleep(2 * time.Second)
}
