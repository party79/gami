package amimock

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NorgannasAddOns/go-uuid"
)

type AmiMockAction func(conn *AmiConn, params textproto.MIMEHeader) textproto.MIMEHeader

//AmiConn for mocking Asterisk AMI
type AmiConn struct {
	context.Context
	cancel    context.CancelFunc
	uuid      string
	srv       *AmiServer
	messageCh chan textproto.MIMEHeader
	connRaw   io.ReadWriteCloser
	conn      *textproto.Conn
}

func NewAmiConn(ctx context.Context, cancel context.CancelFunc, conn io.ReadWriteCloser, c *AmiServer) *AmiConn {
	amic := &AmiConn{
		Context:   ctx,
		cancel:    cancel,
		uuid:      uuid.New("C"),
		srv:       c,
		messageCh: make(chan textproto.MIMEHeader, 100),
		connRaw:   conn,
		conn:      textproto.NewConn(conn),
	}
	go amic.doHeartBeat()
	go amic.doReader()
	go amic.doWriter()
	amic.conn.PrintfLine("Asterisk Call Manager")
	return amic
}

//Emit: emits packet to connection
func (conn *AmiConn) Emit(packet textproto.MIMEHeader) {
	conn.messageCh <- packet
}

func (conn *AmiConn) Close() {
	close(conn.messageCh)
	conn.conn = nil
	conn.connRaw.Close()
	conn.cancel()
}

func (conn *AmiConn) doHeartBeat() {
	for now := range time.Tick(time.Second) {
		if conn.conn == nil {
			break
		}
		conn.messageCh <- textproto.MIMEHeader{
			"Event": {"HeartBeat"},
			"Time":  {strconv.Itoa(int(now.Unix()))},
		}
	}
}
func (conn *AmiConn) doReader() {
	for {
		packet, err := conn.conn.ReadMIMEHeader()
		if conn.conn == nil {
			break
		}
		if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
			continue
		}
		if err != nil {
			errStr := err.Error()
			if len(errStr) > 32 && errStr[len(errStr)-32:len(errStr)] == "use of closed network connection" {
				break
			}
			log.Printf("Socket error: %v", err)
			conn.Close()
			break
		}
		action := packet.Get("Action")
		if action == "Ping" {
			conn.messageCh <- textproto.MIMEHeader{
				"Response": {"Success"},
				"ActionID": {packet.Get("Actionid")},
			}
		} else if cb, ok := conn.srv.getMock(action); ok {
			conn.messageCh <- cb(conn, packet)
		} else {
			conn.messageCh <- textproto.MIMEHeader{
				"Response": {"TEST"},
				"ActionID": {packet.Get("Actionid")},
			}
		}
	}
}
func (conn *AmiConn) doWriter() {
	for {
		packet, x := <-conn.messageCh
		if !x {
			break
		}
		if conn.conn == nil {
			return
		}
		var output string = ""
		for k, v := range packet {
			for _, v2 := range v {
				output = output + fmt.Sprintf("%s: %s\r\n", k, strings.TrimSpace(v2))
			}
		}
		err := conn.conn.PrintfLine("%s", output)
		if conn.conn == nil {
			return
		}
		if err != nil {
			log.Printf("Conn.Write Error: %v", err)
			continue
		}
	}
}

//AmiServer for mocking Asterisk AMI
type AmiServer struct {
	sync.RWMutex
	Addr          string
	actionsMocked map[string]AmiMockAction
	listener      net.Listener
	conns         []*AmiConn
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
		actionsMocked: make(map[string]AmiMockAction),
		conns:         make([]*AmiConn, 0),
	}
	go srv.do(listener)
	return srv
}

//Mock: adds an action to mock
func (c *AmiServer) Mock(action string, cb AmiMockAction) {
	c.Lock()
	defer c.Unlock()
	c.actionsMocked[action] = cb
}

//Unmock: removes an action from mocking
func (c *AmiServer) Unmock(action string) {
	c.Lock()
	defer c.Unlock()
	delete(c.actionsMocked, action)
}

//Clear: clears all actions from mocking
func (c *AmiServer) Clear() {
	c.Lock()
	defer c.Unlock()
	c.actionsMocked = make(map[string]AmiMockAction)
}

//Emit: emits packet to all connections
func (c *AmiServer) Emit(packet textproto.MIMEHeader) {
	c.RLock()
	defer c.RUnlock()
	for _, conn := range c.conns {
		conn.Emit(packet)
	}
}

func (c *AmiServer) getMock(action string) (AmiMockAction, bool) {
	c.RLock()
	defer c.RUnlock()
	if cb, ok := c.actionsMocked[action]; ok {
		return cb, true
	} else if cb, ok := c.actionsMocked["default"]; ok {
		return cb, true
	}
	return nil, false
}

func (c *AmiServer) do(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		func(conn net.Conn) {
			var (
				ctx    context.Context
				cancel context.CancelFunc
			)
			ctx, cancel = context.WithCancel(context.Background())
			amic := NewAmiConn(ctx, cancel, conn, c)
			c.Lock()
			defer c.Unlock()
			c.conns = append(c.conns, amic)
			go func(amic *AmiConn) {
				_, _ = <-amic.Done()
				c.Lock()
				defer c.Unlock()
				repl := make([]*AmiConn, 0)
				for _, conn := range c.conns {
					if conn.uuid != amic.uuid {
						repl = append(repl, conn)
					}
				}
				c.conns = repl
			}(amic)
		}(conn)
	}
}

func (c *AmiServer) CloseCons() {
	c.Lock()
	defer c.Unlock()
	for _, conn := range c.conns {
		conn.Close()
	}
	c.conns = make([]*AmiConn, 0)
}
func (c *AmiServer) Close() {
	c.CloseCons()
	c.listener.Close()
}
