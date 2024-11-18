package jsonrpc

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
)

type wait struct {
	keep     bool
	response func(json.RawMessage)
}

// ReadWriteCloser implements all methods of io.Reader, io.Writer, and io.Closer.
type ReadWriteCloser interface {
	io.ReadWriter
	io.Closer
}

type clientResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

type clientRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// Client represents a client connection to a JSONRPC server.
type Client struct {
	clientHandler
}

type clientHandler struct {
	encoder *json.Encoder
	decoder *json.Decoder
	closer  io.Closer

	mu       sync.Mutex
	nextID   int
	requests map[int]chan clientResponse
	waits    map[int]*wait
}

// NewClient create a new client from the given connection.
func NewClient(rw ReadWriteCloser) *Client {
	c := &Client{
		clientHandler: clientHandler{
			decoder:  json.NewDecoder(rw),
			closer:   rw,
			encoder:  json.NewEncoder(rw),
			requests: make(map[int]chan clientResponse),
			waits:    make(map[int]*wait),
		},
	}

	go c.respond()

	return c
}

func (c *Client) respond() {
	for {
		var resp clientResponse

		if err := c.decoder.Decode(&resp); err != nil {
			return
		}

		c.handleResponse(resp)
	}
}

func (c *clientHandler) handleResponse(resp clientResponse) {
	c.mu.Lock()

	if resp.ID >= 0 {
		if ch, ok := c.requests[resp.ID]; ok {
			delete(c.requests, resp.ID)
			ch <- resp
		}
	} else {
		if w, ok := c.waits[resp.ID]; ok {
			if !w.keep {
				delete(c.waits, resp.ID)
			}

			go w.response(resp.Result)
		}
	}

	c.mu.Unlock()
}

// Request makes an RPC call to the connected server with the given method and
// params.
//
// The params will be JSON encoded.
//
// Returns the JSON encoded response from the server, or an error.
func (c *clientHandler) Request(method string, params any) (json.RawMessage, error) {
	ch := make(chan clientResponse)

	c.mu.Lock()

	id := c.nextID
	c.nextID++
	c.requests[id] = ch

	c.mu.Unlock()

	c.encoder.Encode(clientRequest{
		ID:     id,
		Method: method,
		Params: params,
	})

	resp := <-ch

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Result, nil
}

// RequestValue acts as Request, but will unmarshal the response into the given
// value.
func (c *clientHandler) RequestValue(method string, params any, response any) error {
	respData, err := c.Request(method, params)
	if err != nil {
		return err
	}

	return json.Unmarshal(respData, response)
}

// Await will wait for a message pushed from the server with the given ID and
// call the given func with the JSON encoded data.
//
// The id given should be a negative value.
func (c *clientHandler) Await(id int, cb func(json.RawMessage)) error {
	return c.wait(id, cb, false)
}

// Subscribe will wait for all messages pushed from the server with the given
// ID and call the given func with the JSON encoded data for each one.
//
// The id given should be a negative value.
func (c *clientHandler) Subscribe(id int, cb func(json.RawMessage)) error {
	return c.wait(id, cb, true)
}

func (c *clientHandler) wait(id int, cb func(json.RawMessage), keep bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.waits[id]; ok {
		return ErrExisting
	}

	c.waits[id] = &wait{
		keep:     keep,
		response: cb,
	}

	return nil
}

// Close will stop all client goroutines and close the connection to the server.
func (c *clientHandler) Close() error {
	c.mu.Lock()

	for _, r := range c.requests {
		r <- clientResponse{
			Error: &Error{
				Message: "conn closed",
			},
		}
	}

	c.mu.Unlock()

	return c.closer.Close()
}

// Error.
var (
	ErrExisting = errors.New("existing waiter")
)
