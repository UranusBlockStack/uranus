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

package utils

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/params"
	urpc "github.com/UranusBlockStack/uranus/rpc"
	"github.com/UranusBlockStack/uranus/rpcapi"
	jww "github.com/spf13/jwalterweatherman"
)

var (
	CoreURL        string
	DefaultCoreURL = utils.EnvString("URANUS_URL", "http://localhost:8000")
	urlPrefix      = "http://"
	OneLine        bool // Streamline pattern, Output less and better content
)

// MustRPCClient Wraper rpc's client
func MustRPCClient() *urpc.Client {
	if !strings.HasPrefix(CoreURL, urlPrefix) {
		CoreURL = urlPrefix + CoreURL
	}

	client, err := urpc.DialHTTP(CoreURL)
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

func PrintJSON(data interface{}) {
	rawData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
	jww.FEEDBACK.Println(string(rawData))
}

func PrintJSONList(data interface{}) {
	value := reflect.ValueOf(data)

	if value.Kind() != reflect.Slice {
		jww.ERROR.Printf("invalid type %v assertion", value.Kind())
		os.Exit(1)
	}

	for idx := 0; idx < value.Len(); idx++ {
		if OneLine {
			s := fmt.Sprint(idx, ":")
			rawData, err := json.MarshalIndent(value.Index(idx).Interface(), "", "  ")
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(1)
			}
			jww.FEEDBACK.Println(s, string(rawData))
		} else {
			jww.FEEDBACK.Println(idx, ":")
			rawData, err := json.MarshalIndent(value.Index(idx).Interface(), "", "  ")
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(1)
			}
			jww.FEEDBACK.Println(string(rawData))
		}
	}
}

func IsHexAddr(str string) string {
	if !utils.IsHexAddr(str) {
		jww.ERROR.Printf("Invalid hex value for address ")
		os.Exit(1)
	}
	return str
}

func IsHexHash(str string) string {
	if !utils.IsHexHash(str) {
		jww.ERROR.Printf("Invalid hex value for hash ")
		os.Exit(1)
	}
	return str
}

func GetBlockheight(arg string) *rpcapi.BlockHeight {
	bh := new(rpcapi.BlockHeight)
	if err := bh.UnmarshalJSON([]byte(arg)); err != nil {
		jww.ERROR.Printf("Invalid fulltx value: %v err: %v", arg, err)
		os.Exit(1)
	}
	return bh
}

func GetUint64(arg string) uint64 {
	num, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		jww.ERROR.Printf("Invalid fulltx value: %v err: %v", arg, err)
		os.Exit(1)
	}
	return num
}

func Getbig(arg string) *big.Int {
	num, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		jww.ERROR.Printf("Invalid fulltx value: %v err: %v", arg, err)
		os.Exit(1)
	}
	return big.NewInt(num)
}

func Version() {
	fmt.Println(strings.Title(params.Identifier))
	gitCommit := params.GitCommit()
	fmt.Println("Version:", params.VersionWithCommit(gitCommit))
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
}

func DefaultDataDir() string {
	home := HomeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "uranus_dir")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "uranus_dir")
		} else {
			return filepath.Join(home, ".uranus_dir")
		}
	}
	return ""
}

func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
