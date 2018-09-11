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
	"os"
	"reflect"

	"github.com/UranusBlockStack/uranus/common/utils"
	urpc "github.com/UranusBlockStack/uranus/rpc"
	jww "github.com/spf13/jwalterweatherman"
)

var (
	coreURL = utils.EnvString("Uranus_URL", "http://localhost:8000")
)

// MustRPCClient Wraper rpc's client
func MustRPCClient() *urpc.Client {
	utils.EnvParse()
	client, err := urpc.DialHTTP("http://127.0.0.1:8000")
	if err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
	return client
}

// ClientCall Wrapper rpc call api.
func ClientCall(path string, args, reply interface{}) {
	client := MustRPCClient()
	err := client.Call(path, args, reply)
	if err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

func printJSON(data interface{}) {
	rawData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
	jww.FEEDBACK.Println(string(rawData))
}

func printJSONList(data interface{}) {

	value := reflect.ValueOf(data)

	if value.Kind() != reflect.Slice {
		jww.ERROR.Printf("invalid type %v assertion", value.Kind())
		os.Exit(1)
	}

	for idx := 0; idx < value.Len(); idx++ {
		jww.FEEDBACK.Println(idx, ":")
		rawData, err := json.MarshalIndent(value.Index(idx).Interface(), "", "  ")
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(1)
		}
		jww.FEEDBACK.Println(string(rawData))
	}
}
