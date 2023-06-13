// Package jsonrpc implements simple JSON RPC client/server message handling systems
package jsonrpc // import "vimagination.zapto.org/jsonrpc"

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

type request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     json.RawMessage `json:"id"`
}

// Response represents a response to a client
type Response struct {
	ID     int    `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Error represents the error type for RPC requests
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Is(target error) bool {
	err, ok := target.(Error)
	if ok {
		return e.Is(&err)
	}

	errr, ok := target.(*Error)
	if !ok {
		return false
	}

	if (errr == nil) != (e == nil) {
		return false
	}

	return errr.Code == e.Code && reflect.DeepEqual(errr.Data, e.Data) && errr.Message == e.Message
}

func (e Error) Error() string {
	return e.Message
}

// Handler takes a method name and a JSON Raw Message byte slice and should
// return data OR an error, not both
type Handler interface {
	HandleRPC(method string, data json.RawMessage) (any, error)
}

// HandlerFunc is a convenience type to wrap a function for the Handler
// interface
type HandlerFunc func(string, json.RawMessage) (any, error)

// HandleRPC implements the Handler inteface
func (r HandlerFunc) HandleRPC(method string, data json.RawMessage) (any, error) {
	return r(method, data)
}

// Server represents a RPC server connection that will handle responses from a
// single client
type Server struct {
	handler Handler
	decoder *json.Decoder

	encoder *json.Encoder
	writer  io.Writer
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
	result, err := s.handler.HandleRPC(req.Method, req.Params)
	return s.send(req.ID, result, err)
}

// Send sends the encoded Response to the client
func (s *Server) Send(resp Response) error {
	return s.encoder.Encode(resp)
}

const (
	jsonHead = "{\"id\":"
	jsonMid  = ",\"result\":"
	jsonErr  = ",\"error\":"
	jsonTail = '}'
)

var jsonNil = json.RawMessage{'n', 'u', 'l', 'l'}

func (s *Server) send(id json.RawMessage, data any, e error) error {
	var (
		err error
		rm  json.RawMessage
		ok  bool
	)
	mid := jsonMid
	if e != nil {
		if errr, ok := e.(*Error); ok {
			rm, err = json.Marshal(errr)
		} else {
			rm, err = json.Marshal(Error{
				Message: e.Error(),
				Data:    e,
			})
		}
		mid = jsonErr
	} else if data == nil {
		rm = jsonNil
	} else {
		rm, ok = data.(json.RawMessage)
		if !ok {
			rm, err = json.Marshal(data)
		} else if len(rm) == 0 {
			rm = jsonNil
		}
	}
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	if _, err = s.writer.Write(append(append(append(append(append(make([]byte, 0, len(jsonHead)+len(id)+len(mid)+len(rm)+1), jsonHead...), id...), mid...), rm...), jsonTail)); err != nil {
		return fmt.Errorf("error writing to socket: %w", err)
	}
	return nil
}

// SendData sends the raw bytes (unencoded) to the client
func (s *Server) SendData(data json.RawMessage) error {
	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("error sending data: %w", err)
	}
	return nil
}
