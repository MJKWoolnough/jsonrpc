package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
)

// ClientServer represents both a client and server RPC connection.
type ClientServer struct {
	serverHandler
	clientHandler
	decoder *json.Decoder
}

// NewClientServer creates both client and server handling on the same
// connection.
func NewClientServer(conn io.ReadWriter, handler Handler) *ClientServer {
	closer, ok := conn.(io.Closer)
	if !ok {
		closer = io.NopCloser(conn)
	}

	cs := &ClientServer{
		serverHandler: serverHandler{
			handler: handler,
			writer:  conn,
		},
		clientHandler: clientHandler{
			closer:   closer,
			encoder:  json.NewEncoder(conn),
			decoder:  json.NewDecoder(conn),
			requests: make(map[int]chan clientResponse),
			waits:    make(map[int]*wait),
		},
	}

	return cs
}

type requestOrResponse struct {
	request
	Result json.RawMessage `json:"result"`
	Error  *Error          `json:"error"`
}

// Handle starts the server's handling loop. This method must be active to
// handle client responses.
//
// The func will return only when it encounters a read error, be it from a
// closed connection, or from some fault on the wire.
func (c *ClientServer) Handle() error {
	for {
		var req requestOrResponse

		if err := c.decoder.Decode(&req); err != nil {
			return fmt.Errorf("error decoding JSON request: %w", err)
		}

		if req.Method != "" {
			go c.serverHandler.handleRequest(req.request)
		} else {
			var id int

			json.Unmarshal(req.ID, &id)

			go c.clientHandler.handleResponse(clientResponse{
				ID:     id,
				Result: req.Result,
				Error:  req.Error,
			})
		}
	}
}
