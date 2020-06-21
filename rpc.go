// Package jsonrpc implements simple JSON RPC client/server message handling systems
package jsonrpc // import "vimagination.zapto.org/jsonrpc"

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
)

type request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     int             `json:"id"`
}

// Response represents a response to a client
type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Handler takes a method name and a JSON Raw Message byte slice and should
// return data OR an error, not both
type Handler interface {
	HandleRPC(method string, data json.RawMessage) (interface{}, error)
}

// HandlerFunc is a convenience type to wrap a function for the Handler
// interface
type HandlerFunc func(string, json.RawMessage) (interface{}, error)

// HandleRPC implements the Handler inteface
func (r HandlerFunc) HandleRPC(method string, data json.RawMessage) (interface{}, error) {
	return r(method, data)
}

// Server represents a RPC server connection that will handle responses from a
// single client
type Server struct {
	handler Handler
	decoder *json.Decoder

	encoderLock sync.Mutex
	encoder     *json.Encoder
	writer      io.Writer
}

// New creates a new Server connection
func New(conn io.ReadWriter, handler Handler) *Server {
	return &Server{
		handler: handler,
		decoder: json.NewDecoder(conn),
		encoder: json.NewEncoder(conn),
		writer:  conn,
	}
}

// Handle starts the server's handling loop.
//
// The func will return only when it encounters a read error, be it from a
// closed connection, or from some fault on the wire.
func (s *Server) Handle() error {
	for {
		var req request
		if err := s.decoder.Decode(&req); err != nil {
			return fmt.Errorf("error decoding JSON request: %w", err)
		}
		go s.handleRequest(req)
	}
}

func (s *Server) handleRequest(req request) error {
	resp := Response{ID: req.ID}
	var err error
	resp.Result, err = s.handler.HandleRPC(req.Method, req.Params)
	if err != nil {
		resp.Error = err.Error()
	}
	return s.Send(resp)
}

// Send sends the encoded Response to the client
func (s *Server) Send(resp Response) error {
	if rm, ok := resp.Result.(json.RawMessage); ok && resp.Error == "" {
		return s.sendJSON(resp.ID, rm)
	}
	s.encoderLock.Lock()
	err := s.encoder.Encode(resp)
	s.encoderLock.Unlock()
	return err
}

var (
	jsonHead = []byte("(\"id\":")
	jsonMid  = []byte(",\"result\":")
	jsonTail = []byte{'}'}
)

func (s *Server) sendJSON(id int, data json.RawMessage) error {
	var err error
	s.encoderLock.Lock()
	if _, err = s.writer.Write(jsonHead); err == nil {
		if _, err = s.writer.Write(strconv.AppendInt(make([]byte, 0, 20), int64(id), 10)); err == nil {
			if _, err = s.writer.Write(jsonMid); err == nil {
				if _, err = s.writer.Write(data); err == nil {
					_, err = s.writer.Write(jsonTail)
				}
			}
		}
	}
	s.encoderLock.Unlock()
	return err
}

// SendData sends the raw bytes (unencoded) to the client
func (s *Server) SendData(data json.RawMessage) error {
	s.encoderLock.Lock()
	_, err := s.writer.Write(data)
	s.encoderLock.Unlock()
	return err
}
