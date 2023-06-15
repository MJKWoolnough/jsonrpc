package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"
)

type rw struct {
	io.ReadCloser
	io.WriteCloser
}

func (r *rw) Close() error {
	r.ReadCloser.Close()
	r.WriteCloser.Close()

	return nil
}

func makeServerClientConn() (ReadWriteCloser, ReadWriteCloser) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()

	return &rw{
			ReadCloser:  ar,
			WriteCloser: aw,
		}, &rw{
			ReadCloser:  br,
			WriteCloser: bw,
		}
}

type simpleHandler struct{}

var ErrUnknownEndpoint = &Error{
	Message: "unknown endpoint",
}

func (simpleHandler) HandleRPC(method string, data json.RawMessage) (any, error) {
	switch method {
	case "add":
		var toAdd [2]int
		if err := json.Unmarshal(data, &toAdd); err != nil {
			return nil, err
		}

		return toAdd[0] + toAdd[1], nil
	}
	return nil, ErrUnknownEndpoint
}

func TestRequest(t *testing.T) {
	serverConn, clientConn := makeServerClientConn()

	s := New(serverConn, new(simpleHandler))
	go s.Handle()
	defer serverConn.Close()

	c := NewClient(clientConn)
	defer c.Close()

	for n, test := range [...]struct {
		Method   string
		Params   any
		Response json.RawMessage
		Error    error
	}{
		{
			Method: "unknown",
			Error:  ErrUnknownEndpoint,
		},
		{
			Method:   "add",
			Params:   [2]int{5, 6},
			Response: json.RawMessage{'1', '1'},
		},
	} {
		resp, err := c.Request(test.Method, test.Params)
		if !errors.Is(test.Error, err) {
			t.Errorf("test %d: expecting error %s, got %s", n+1, test.Error, err)
		} else if !bytes.Equal(test.Response, resp) {
			t.Errorf("test %d: expecting response %s, got %s", n+1, test.Response, resp)
		}
	}
}

func TestAwait(t *testing.T) {
	t.Parallel()
	serverConn, clientConn := makeServerClientConn()

	s := New(serverConn, new(simpleHandler))
	go s.Handle()
	defer serverConn.Close()

	c := NewClient(clientConn)
	defer c.Close()

	resp := make(chan int, 2)

	c.Await(-1, func(data json.RawMessage) {
		var num int
		json.Unmarshal(data, &num)

		resp <- num
	})

	total := receiveData(s, resp)
	if total != 5 {
		t.Errorf("expecting result 5, got %d", total)
	}
}

func receiveData(s *Server, ch chan int) int {
	s.Send(Response{
		ID:     -1,
		Result: 5,
	})
	s.Send(Response{
		ID:     -1,
		Result: 6,
	})

	var total int
	timeout := time.After(time.Second)

Loop:
	for {
		select {
		case num := <-ch:
			total += num
		case <-timeout:
			break Loop
		}
	}

	return total
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	serverConn, clientConn := makeServerClientConn()

	s := New(serverConn, new(simpleHandler))
	go s.Handle()
	defer serverConn.Close()

	c := NewClient(clientConn)
	defer c.Close()

	resp := make(chan int, 2)

	c.Subscribe(-1, func(data json.RawMessage) {
		var num int
		json.Unmarshal(data, &num)

		resp <- num
	})

	total := receiveData(s, resp)

	if total != 11 {
		t.Errorf("expecting result 11, got %d", total)
	}
}

func TestRequestValue(t *testing.T) {
	serverConn, clientConn := makeServerClientConn()

	s := New(serverConn, new(simpleHandler))
	go s.Handle()
	defer serverConn.Close()

	c := NewClient(clientConn)
	defer c.Close()

	for n, test := range [...]struct {
		Method   string
		Params   any
		Response any
		Error    error
	}{
		{
			Method:   "add",
			Params:   [2]int{5, 6},
			Response: int(11),
		},
	} {
		v := reflect.New(reflect.TypeOf(test.Response))
		err := c.RequestValue(test.Method, test.Params, v.Interface())
		if !errors.Is(test.Error, err) {
			t.Errorf("test %d: expecting error %s, got %s", n+1, test.Error, err)
		} else if resp := v.Elem().Interface(); !reflect.DeepEqual(test.Response, resp) {
			t.Errorf("test %d: expecting response %s, got %s", n+1, test.Response, resp)
		}
	}
}
