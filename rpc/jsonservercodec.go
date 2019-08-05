package rpc

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
)

var errMissingParams = errors.New("jsonrpc: request body missing params")

type jsonServerCodec struct {
	encode func(v interface{}) error // for writing JSON values
	decode func(v interface{}) error // for reading JSON values
	c      io.Closer

	// temporary work space
	req serverRequest

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex      sync.Mutex // protects seq, pending
	seq        uint64
	pending    map[uint64]*json.RawMessage
	remoteAddr string
}

// NewJSONServerCodec returns a new ServerCodec using JSON-RPC on conn.
func NewJSONServerCodec(conn Conn, encode, decode func(v interface{}) error) ServerCodec {
	codec := &jsonServerCodec{
		decode:  decode,
		encode:  encode,
		c:       conn,
		pending: make(map[uint64]*json.RawMessage),
	}

	if ra, ok := conn.(connWithRemoteAddr); ok {
		codec.remoteAddr = ra.RemoteAddr()
	}
	return codec
}

func (c *jsonServerCodec) ReadRequestHeader(r *Request) error {
	c.req.reset()
	if err := c.decode(&c.req); err != nil {
		return err
	}
	r.ServiceMethod = c.req.Method

	// JSON request id can be any JSON value;
	// RPC package expects uint64.  Translate to
	// internal uint64 and save JSON on the side.
	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = c.req.Id
	c.req.Id = nil
	r.Seq = c.seq
	c.mutex.Unlock()

	return nil
}

func (c *jsonServerCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return errMissingParams
	}
	// JSON params is array value.
	// RPC params is struct.
	// Unmarshal into array containing struct for now.
	// Should think about making RPC more general.
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.req.Params, &params)
}

var null = json.RawMessage([]byte("null"))

func (c *jsonServerCodec) WriteResponse(r *Response, x interface{}) error {
	c.mutex.Lock()
	b, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	if b == nil {
		// Invalid request so no id. Use JSON null.
		b = &null
	}
	resp := serverResponse{Id: b}
	if r.Error == "" {
		resp.Result = x
	} else {
		resp.Error = r.Error
	}
	return c.encode(resp)
}

func (c *jsonServerCodec) Close() error {
	return c.c.Close()
}
