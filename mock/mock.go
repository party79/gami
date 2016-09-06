package amimock

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"net/textproto"
	"sync"
	"time"
)

type AmiMockAction func(params textproto.MIMEHeader) textproto.MIMEHeader

//AmiServer for mocking Asterisk AMI
type AmiServer struct {
	Addr          string
	actionsMocked map[string]amiMockAction
	listener      net.Listener
	mu            *sync.RWMutex
}

//NewAmiServer: Creats an ami mock server
func NewAmiServer() *AmiServer {
	addr := "localhost:0"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	srv := &AmiServer{
		Addr:          listener.Addr().String(),
		listener:      listener,
		actionsMocked: make(map[string]amiMockAction),
	}
	go srv.do(listener)
	return srv
}

//Mock: adds an action to mock
func (c *AmiServer) Mock(action string, cb amiMockAction) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.actionsMocked[action] = cb
}

//Unmock: removes an action from mocking
func (c *AmiServer) Unmock(action string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.actionsMocked, action)
}

//Clear: clears all actions from mocking
func (c *AmiServer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.actionsMocked = make(map[string]amiMockAction)
}

func (c *AmiServer) do(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		fmt.Fprintf(conn, "Asterisk Call Manager\r\n")
		tconn := textproto.NewConn(conn)
		//install event HeartBeat
		go func(conn *textproto.Conn) {
			for now := range time.Tick(time.Second) {
				fmt.Fprintf(conn.W, "Event: HeartBeat\r\nTime: %d\r\n\r\n",
					now.Unix())
			}
		}(tconn)

		go func(conn *textproto.Conn) {
			defer conn.Close()

			for {
				header, err := conn.ReadMIMEHeader()
				if err != nil {
					return
				}
				var output bytes.Buffer

				time.AfterFunc(time.Millisecond*time.Duration(rand.Intn(1000)), func() {
					c.mu.RLock()
					defer c.mu.RUnlock()

					if cb, ok := c.actionsMocked[header.Get("Action")]; ok {
						rvals := cb(header)
						for k, vals := range rvals {
							fmt.Fprintf(&output, "%s: %s\r\n", k, vals)
						}
						output.WriteString("\r\n")

						err := conn.PrintfLine(output.String())
						if err != nil {
							panic(err)
						}
					} else if cb, ok := c.actionsMocked["default"]; ok {
						rvals := cb(header)
						for k, vals := range rvals {
							fmt.Fprintf(&output, "%s: %s\r\n", k, vals)
						}
						output.WriteString("\r\n")

						err := conn.PrintfLine(output.String())
						if err != nil {
							panic(err)
						}
					} else {
						//default response
						fmt.Fprintf(&output, "Response: TEST\r\nActionID: %s\r\n\r\n", header.Get("Actionid"))
						err := conn.PrintfLine(output.String())
						if err != nil {
							panic(err)
						}
					}
				})
			}
		}(tconn)
	}
}

func (c *AmiServer) Close() {
	c.listener.Close()
}
