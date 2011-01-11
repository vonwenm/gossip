// Connectionless transport service
package gossip

import (
	"net"
	"os"
	"strconv"
	"fmt"
)

type Message []byte

// See RFC 1035 Section 4.2.1
const MessageSize = 512

// Message with a source or destination address of the form host:port.
type Packet struct {
	Addr *net.UDPAddr
	Msg  Message
}

// Closure to handle incoming packets
type EventHandler func(*Conn, *Packet)

// Once connected, any errors encountered are piped
// down Conn.Err; this channel is closed on disconnect.
type Conn struct {
	// Error channel to transmit any fail back to the caller
	Err chan os.Error

	// Handle incoming packets read from the socket
	handlers []EventHandler

	sock *net.UDPConn
	in   chan *Packet
	out  chan *Packet
}

// Returns a nil packet if the addr cannot be resolved.
func NewPacket(addr string, msg Message) *Packet {
	udpAddr, err := net.ResolveUDPAddr(addr)
	if err != nil {
		return nil
	}
	return &Packet{udpAddr, msg}
}

func NewConn() *Conn {
	conn := new(Conn)
	conn.initialize()
	return conn
}

func (conn *Conn) initialize() {
	conn.in = make(chan *Packet)
	conn.out = make(chan *Packet)
	conn.Err = make(chan os.Error, 4)
	conn.handlers = make([]EventHandler, 0, 4)
	conn.sock = nil
}

var (
	ErrAlreadyConnected = os.NewError("Socket is already open")
	ErrClosedConn       = os.NewError("Socked has been closed")
	ErrNilPacket        = os.NewError("Encountered nil packet")
)

func (conn *Conn) Listen(port uint) (err os.Error) {
	if conn.IsConnected() {
		return ErrAlreadyConnected
	}

	// bind to all IP addresses on the system with the specified port
	var laddr *net.UDPAddr
	if laddr, err = net.ResolveUDPAddr(":" + strconv.Uitoa(port)); err != nil {
		return err
	}

	if conn.sock, err = net.ListenUDP("udp4", laddr); err != nil {
		return err
	}
	conn.spawn()
	return nil
}


func (conn *Conn) Dial(remoteAddr string) (err os.Error) {
	if conn.IsConnected() {
		return ErrAlreadyConnected
	}

	var raddr *net.UDPAddr
	if raddr, err = net.ResolveUDPAddr(remoteAddr); err != nil {
		return err
	}
	if conn.sock, err = net.DialUDP("udp4", nil, raddr); err != nil {
		return err
	}
	conn.spawn()
	return nil
}

func (conn *Conn) IsConnected() bool {
	return conn.sock != nil
}

func (conn *Conn) Disconnect() {
	close(conn.in)
	close(conn.out)
	close(conn.Err)
	conn.sock.Close()

	// be ready for the next connection
	conn.initialize()
}

func (conn *Conn) Send(msg Message) {
	if conn.IsConnected() {
		conn.out <- &Packet{nil, msg}
	} else {
		conn.Err <- ErrClosedConn
	}
}

func (conn *Conn) SendTo(msg Message, addr *net.UDPAddr) {
	if conn.IsConnected() {
		conn.out <- &Packet{addr, msg}
	} else {
		conn.error("conn.SendTo(): Cannot send message to %s because %s",
			addr.String(), ErrClosedConn.String())
	}
}

// Start background processes
func (conn *Conn) spawn() {
	go conn.sending()
	go conn.dispatching()
	go conn.receiving()
}

// Keep on writing outgoing messages to the socket
func (conn *Conn) sending() {
	for p := range conn.out {
		if p == nil {
			conn.Err <- ErrNilPacket
			continue
		}

		var err os.Error
		if p.Addr == nil {
			if _, err = conn.sock.Write(p.Msg); err != nil {
				conn.error("conn.sending(): %s", err.String())
			}
		} else {
			if _, err = conn.sock.WriteTo(p.Msg, p.Addr); err != nil {
				conn.error("conn.sending() [%s]: %s", p.Addr.String(), err.String())
			}
		}
		if err != nil {
			conn.Disconnect()
			break
		}

	}
}

// Keep on reading incoming packets from the socket
func (conn *Conn) receiving() {
	buff := makeMessage()
	for {
		msgSize, addr, err := conn.sock.ReadFrom(buff)
		if err != nil {
			conn.error("conn.receiving(): %s", err.String())
			conn.Disconnect()
			break
		}

		msg := make(Message, msgSize)
		copy(msg, buff)
		udpAddr, _ := addr.(*net.UDPAddr)
		conn.in <- &Packet{udpAddr, msg}
	}
}

// Keep on dispatching incoming packets to event handlers
func (conn *Conn) dispatching() {
	for p := range conn.in {
		conn.dispatchEvent(p)
	}
}

// Loops through all event handlers and dispatches an incoming packet to them.
// Each event handler are run in its own goroutine.
func (conn *Conn) dispatchEvent(p *Packet) {
	for _, f := range conn.handlers {
		go f(conn, p)
	}
}

func (conn *Conn) AddHandler(f EventHandler) {
	conn.handlers = append(conn.handlers, f)
}

// Allocate memory for a new Message with a capacity of MessageSize
func makeMessage() Message {
	return make(Message, MessageSize)
}

// Dispatch a human-readable os.Error to the error channel.
func (conn *Conn) error(s string, a ...interface{}) {
	conn.Err <- os.NewError(fmt.Sprintf(s, a...))
}
