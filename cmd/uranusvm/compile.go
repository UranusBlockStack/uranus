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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	jww "github.com/spf13/jwalterweatherman"
)

type solCompileOutput struct {
	HexByteCodes    string
	FunctionHashMap map[string]solMethod
}

func (output *solCompileOutput) EncodeRLP(w io.Writer) error {
	value := make([][]byte, 0)

	for k, v := range output.FunctionHashMap {
		vb, err := rlp.Serialize(v)
		if err != nil {
			return err
		}
		value = append(value, []byte(k), vb)
	}

	return rlp.Encode(w, value)
}

func (output *solCompileOutput) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}

	var kvs [][]byte
	if err := rlp.Deserialize(raw, &kvs); err != nil {
		return err
	}

	output.FunctionHashMap = make(map[string]solMethod)

	for i, len := 0, len(kvs)/2; i < len; i++ {
		key := string(kvs[2*i])
		value := kvs[2*i+1]

		m := solMethod{}
		if err := rlp.Deserialize(value, &m); err != nil {
			return err
		}

		output.FunctionHashMap[key] = m
	}

	return nil
}

// compile compiles the specified solidity file and returns the compilation outputs
// and dispose method to clear the compilation resources. Returns nil if any error occurrred.
func compile(solFilePath, solcPath string) (*solCompileOutput, func()) {
	if is, err := utils.IsDirExist(solFilePath); err != nil && !is {
		jww.ERROR.Println("The specified solidity file does not exist,", solFile, err)
		return nil, nil
	}

	// output to temp dir
	tempDir, err := ioutil.TempDir("", "SolCompile-")
	if err != nil {
		jww.ERROR.Println("Failed to create temp folder for solidity compilation,", err.Error())
		return nil, nil
	}

	deleteTempDir := true
	defer func() {
		if deleteTempDir {
			os.RemoveAll(tempDir)
		}
	}()

	// run solidity compilation command
	cmdArgs := fmt.Sprintf("--optimize --bin --hashes -o %v %v", tempDir, solFile)
	cmd := exec.Command(solcPath, strings.Split(cmdArgs, " ")...)
	if err = cmd.Run(); err != nil {
		jww.ERROR.Println("Failed to compile the solidity file,", err.Error())
		return nil, nil
	}

	// walk through the temp dir to construct compilation outputs
	output := new(solCompileOutput)
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		switch filepath.Ext(path) {
		case ".signatures":
			output.parseFuncHash(string(content))
		case ".bin":
			output.HexByteCodes = ensurePrefix(string(content), "0x")
		}

		return nil
	}

	if err = filepath.Walk(tempDir, walkFunc); err != nil {
		jww.ERROR.Println("Failed to walk through the compilation temp folder,", err.Error())
		return nil, nil
	}

	deleteTempDir = false

	return output, func() {
		os.RemoveAll(tempDir)
	}
}

func (output *solCompileOutput) parseFuncHash(content string) {
	output.FunctionHashMap = make(map[string]solMethod)

	for _, line := range strings.Split(content, "\n") {
		// line: funcHash: method(type1,type2,...)
		if line = strings.Trim(line, "\r"); len(line) == 0 {
			continue
		}

		// add mapping: funcFullName <-> hash
		funcHash := line[:8]
		funcFullName := string(line[10:])
		method := newSolMethod(funcFullName, funcHash)

		output.FunctionHashMap[funcFullName] = method
	}
}

func (output *solCompileOutput) getMethodByName(name string) *solMethod {
	if v, ok := output.FunctionHashMap[name]; ok {
		return &v
	}

	var method solMethod

	for _, v := range output.FunctionHashMap {
		if v.ShortName != name {
			continue
		}

		if len(method.ShortName) > 0 {
			jww.ERROR.Println("Short method name not supported for overloaded methods:")
			output.funcHashesUsage()
			return nil
		}

		method = v
	}

	if len(method.ShortName) == 0 {
		jww.ERROR.Println("Cannot find the specified method name, please call below methods:")
		output.funcHashesUsage()
	}

	return &method
}

func (output *solCompileOutput) funcHashesUsage() {
	for k := range output.FunctionHashMap {
		fmt.Printf("\t%v\n", k)
	}
}
