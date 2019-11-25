# jsonrpc
--
    import "vimagination.zapto.org/jsonrpc"


## Usage

#### type Handler

```go
type Handler interface {
	HandleRPC(method string, data []byte) (interface{}, error)
}
```

Handler takes a method name and a byte slice representing JSON encoded data and
should return data OR an error

#### type HandlerFunc

```go
type HandlerFunc func(string, []byte) (interface{}, error)
```

HandlerFunc is a convenience type to wrap a function for the Handler interface

#### func (HandlerFunc) HandleRPC

```go
func (r HandlerFunc) HandleRPC(method string, data []byte) (interface{}, error)
```
HandleRPC implements the Handler inteface

#### type Response

```go
type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
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
func New(conn io.ReadWriter, handler RPCHandler) *Server
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
func (s *Server) Send(resp Response)
```
Send sends the encoded Response to the client

#### func (*Server) SendData

```go
func (s *Server) SendData(data []byte)
```
SendData sends the raw bytes (unencoded) to the client
