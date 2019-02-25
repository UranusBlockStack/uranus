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
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/rs/cors"
)

const (
	maxHTTPRequestContentLength = 1024 * 128
)

type httpReadWriteNopCloser struct {
	io.Reader
	io.Writer
}

func (c *httpReadWriteNopCloser) Close() error { return nil }

// NewHTTPServer creates a new HTTP RPC server around an API provider.
func NewHTTPServer(srv *Server, cors []string) *http.Server {
	return &http.Server{Handler: newCorsHandler(srv, cors)}
}

// Client represents a JSON-RPC client.
type httpClient struct {
	client     *http.Client
	req        *http.Request
	resp       chan *http.Response
	remainResp *http.Response
	canRead    bool
}

// DialHTTP creates a new RPC clients that connection to an RPC server over HTTP.
func DialHTTP(url string) (*Client, error) {
	client := new(http.Client)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return NewClient(&httpClient{client, req, make(chan *http.Response), &http.Response{}, false}), nil
}

// Write implements io.Writer interface.
func (c *httpClient) Write(d []byte) (n int, err error) {
	c.req.ContentLength = int64(len(d))
	c.req.Body = ioutil.NopCloser(bytes.NewReader(d))
	resp, err := c.client.Do(c.req)
	if err != nil {
		return 0, err
	}
	c.resp <- resp
	return len(d), nil
}

// Read implements io.Reader interface.
func (c *httpClient) Read(p []byte) (n int, err error) {
	var resp *http.Response

	if !c.canRead {
		resp = <-c.resp
	} else {
		resp = c.remainResp
	}

	n, err = resp.Body.Read(p)
	if err != nil {
		defer resp.Body.Close()
		c.canRead = false
	} else {
		c.remainResp = resp
		c.canRead = true
	}

	return n, err
}

// Close implements io.Closer interface.
func (c *httpClient) Close() error {
	c.req.Body.Close()
	return nil
}

func newCorsHandler(srv *Server, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return srv
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(srv)
}
