package amimock

import (
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
	"golang.org/x/net/context"
)

type AmiMocker struct {
	super  *AmiMocker
	action AmiAdvMockAction
}

func (mocker *AmiMocker) Call(conn *AmiConn, params textproto.MIMEHeader) textproto.MIMEHeader {
	if mocker == nil {
		return textproto.MIMEHeader{}
	}
	return mocker.action(conn, params, mocker.super)
}

type AmiMockAction func(conn *AmiConn, params textproto.MIMEHeader) textproto.MIMEHeader
type AmiAdvMockAction func(conn *AmiConn, params textproto.MIMEHeader, super *AmiMocker) textproto.MIMEHeader

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

func NewAmiConn(ctx context.Context, conn io.ReadWriteCloser, c *AmiServer) *AmiConn {
	var (
		sctx   context.Context
		cancel context.CancelFunc
	)
	sctx, cancel = context.WithCancel(ctx)
	amic := &AmiConn{
		Context:   sctx,
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
	go func(amic *AmiConn) {
		<-amic.Done()
		amic.Close()
	}(amic)
	amic.conn.PrintfLine("Asterisk Call Manager")
	return amic
}

//Emit: emits packet to connection
func (conn *AmiConn) Emit(packet textproto.MIMEHeader) {
	if conn.conn == nil {
		return
	}
	conn.messageCh <- packet
}

func (conn *AmiConn) Close() {
	if conn.conn == nil {
		return
	}
	conn.conn = nil
	close(conn.messageCh)
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
			if len(errStr) > 32 && errStr[len(errStr)-32:] == "use of closed network connection" {
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
		} else if mocker, ok := conn.srv.getMock(action); ok {
			conn.messageCh <- mocker.Call(conn, packet)
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
	context.Context
	cancel        context.CancelFunc
	Addr          string
	actionsMocked map[string]*AmiMocker
	listener      net.Listener
	conns         []*AmiConn
}

//NewAmiServer: Creats an ami mock server
func NewAmiServer(ctx context.Context) *AmiServer {
	addr := "localhost:0"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	var (
		sctx   context.Context
		cancel context.CancelFunc
	)
	sctx, cancel = context.WithCancel(ctx)
	srv := &AmiServer{
		Context:       sctx,
		cancel:        cancel,
		Addr:          listener.Addr().String(),
		listener:      listener,
		actionsMocked: make(map[string]*AmiMocker),
		conns:         make([]*AmiConn, 0),
	}
	go srv.do()
	go func(srv *AmiServer) {
		<-srv.Done()
		srv.Close()
	}(srv)
	return srv
}

//AdvMock: adds an action to mock with super support
func (c *AmiServer) AdvMock(action string, cb AmiAdvMockAction) {
	c.Lock()
	defer c.Unlock()
	mocker := &AmiMocker{action: cb}
	if v, ok := c.actionsMocked[action]; ok {
		mocker.super = v
	}
	c.actionsMocked[action] = mocker
}

//Mock: adds an action to mock
func (c *AmiServer) Mock(action string, cb AmiMockAction) {
	c.Lock()
	defer c.Unlock()
	mocker := &AmiMocker{
		action: func(conn *AmiConn, params textproto.MIMEHeader, super *AmiMocker) textproto.MIMEHeader {
			return cb(conn, params)
		},
	}
	if v, ok := c.actionsMocked[action]; ok {
		mocker.super = v
	}
	c.actionsMocked[action] = mocker
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
	c.actionsMocked = make(map[string]*AmiMocker)
}

//Emit: emits packet to all connections
func (c *AmiServer) Emit(packet textproto.MIMEHeader) {
	c.RLock()
	defer c.RUnlock()
	for _, conn := range c.conns {
		conn.Emit(packet)
	}
}

func (c *AmiServer) getMock(action string) (*AmiMocker, bool) {
	c.RLock()
	defer c.RUnlock()
	if mocker, ok := c.actionsMocked[action]; ok {
		return mocker, true
	} else if mocker, ok := c.actionsMocked["default"]; ok {
		return mocker, true
	}
	return nil, false
}

func (c *AmiServer) do() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			c.cancel()
			return
		}
		go func(c *AmiServer, conn net.Conn) {
			amic := NewAmiConn(c.Context, conn, c)
			c.Lock()
			defer c.Unlock()
			c.conns = append(c.conns, amic)
			go func(c *AmiServer, amic *AmiConn) {
				<-amic.Done()
				c.Lock()
				defer c.Unlock()
				repl := make([]*AmiConn, 0)
				for _, conn := range c.conns {
					if conn.uuid != amic.uuid {
						repl = append(repl, conn)
					}
				}
				c.conns = repl
			}(c, amic)
		}(c, conn)
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
	c.cancel()
}
