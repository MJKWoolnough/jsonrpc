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

type Client struct {
	encoder *json.Encoder
	decoder *json.Decoder
	closer  io.Closer
	nextID  int

	mu       sync.Mutex
	requests map[int]chan clientResponse
	waits    map[int]*wait
}

func NewClient(rw ReadWriteCloser) *Client {
	c := &Client{
		encoder:  json.NewEncoder(rw),
		decoder:  json.NewDecoder(rw),
		closer:   rw,
		requests: make(map[int]chan clientResponse),
		waits:    make(map[int]*wait),
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

		c.mu.Lock()
		if resp.ID >= 0 {
			ch, ok := c.requests[resp.ID]
			if ok {
				delete(c.requests, resp.ID)
				ch <- resp
			}
		} else {
			w, ok := c.waits[resp.ID]
			if ok {
				if !w.keep {
					delete(c.waits, resp.ID)
				}
				go w.response(resp.Result)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Client) Request(method string, params any) (json.RawMessage, error) {
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

func (c *Client) RequestValue(method string, params any, response any) error {
	respData, err := c.Request(method, params)
	if err != nil {
		return err
	}

	return json.Unmarshal(respData, response)
}

func (c *Client) Await(id int, cb func(json.RawMessage)) error {
	return c.wait(id, cb, false)
}

func (c *Client) Subscribe(id int, cb func(json.RawMessage)) error {
	return c.wait(id, cb, true)
}

func (c *Client) wait(id int, cb func(json.RawMessage), keep bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.waits[id]
	if ok {
		return ErrExisting
	}
	c.waits[id] = &wait{
		keep:     keep,
		response: cb,
	}
	return nil
}

func (c *Client) Close() error {
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

var ErrExisting = errors.New("existing waiter")
