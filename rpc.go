// Package jsonrpc implements simple JSON RPC client/server message handling systems
package jsonrpc // import "vimagination.zapto.org/jsonrpc"

import (
	"encoding/json"
	"fmt"
	"io"
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

// Handler takes a method name and a byte slice representing JSON encoded
// data and should return data OR an error
type Handler interface {
	HandleRPC(method string, data []byte) (interface{}, error)
}

// HandlerFunc is a convenience type to wrap a function for the Handler
// interface
type HandlerFunc func(string, []byte) (interface{}, error)

// HandleRPC implements the Handler inteface
func (r HandlerFunc) HandleRPC(method string, data []byte) (interface{}, error) {
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

func (s *Server) handleRequest(req request) {
	resp := Response{ID: req.ID}
	var err error
	resp.Result, err = s.handler.HandleRPC(req.Method, req.Params)
	if err != nil {
		resp.Error = err.Error()
	}
	s.Send(resp)
}

// Send sends the encoded Response to the client
func (s *Server) Send(resp Response) {
	s.encoderLock.Lock()
	s.encoder.Encode(resp)
	s.encoderLock.Unlock()
}

// SendData sends the raw bytes (unencoded) to the client
func (s *Server) SendData(data []byte) {
	s.encoderLock.Lock()
	s.writer.Write(data)
	s.encoderLock.Unlock()
}
