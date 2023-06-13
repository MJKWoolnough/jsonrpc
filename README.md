# jsonrpc
--
    import "vimagination.zapto.org/jsonrpc"

Package jsonrpc implements simple JSON RPC client/server message handling
### systems

## Usage

```go
var ErrExisting = errors.New("existing waiter")
```

#### type Client

```go
type Client struct {
}
```


#### func  NewClient

```go
func NewClient(rw ReadWriteCloser) *Client
```

#### func (*Client) Await

```go
func (c *Client) Await(id int, cb func(json.RawMessage)) error
```

#### func (*Client) Close

```go
func (c *Client) Close() error
```

#### func (*Client) Request

```go
func (c *Client) Request(method string, params any) (json.RawMessage, error)
```

#### func (*Client) Subscribe

```go
func (c *Client) Subscribe(id int, cb func(json.RawMessage)) error
```

#### type Error

```go
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
```

Error represents the error type for RPC requests

#### func (Error) Error

```go
func (e Error) Error() string
```

#### func (*Error) Is

```go
func (e *Error) Is(target error) bool
```

#### type Handler

```go
type Handler interface {
	HandleRPC(method string, data json.RawMessage) (any, error)
}
```

Handler takes a method name and a JSON Raw Message byte slice and should return
data OR an error, not both

#### type HandlerFunc

```go
type HandlerFunc func(string, json.RawMessage) (any, error)
```

HandlerFunc is a convenience type to wrap a function for the Handler interface

#### func (HandlerFunc) HandleRPC

```go
func (r HandlerFunc) HandleRPC(method string, data json.RawMessage) (any, error)
```
HandleRPC implements the Handler inteface

#### type ReadWriteCloser

```go
type ReadWriteCloser interface {
	io.ReadWriter
	io.Closer
}
```


#### type Response

```go
type Response struct {
	ID     int    `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}
```

Response represents a response to a client

#### type Server

```go
type Server struct {
}
```

Server represents a RPC server connection that will handle responses from a
single client

#### func  New

```go
func New(conn io.ReadWriter, handler Handler) *Server
```
New creates a new Server connection

#### func (*Server) Handle

```go
func (s *Server) Handle() error
```
Handle starts the server's handling loop.

The func will return only when it encounters a read error, be it from a closed
connection, or from some fault on the wire.

#### func (*Server) Send

```go
func (s *Server) Send(resp Response) error
```
Send sends the encoded Response to the client

#### func (*Server) SendData

```go
func (s *Server) SendData(data json.RawMessage) error
```
SendData sends the raw bytes (unencoded) to the client
