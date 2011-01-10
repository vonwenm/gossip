package transport

import (
	"testing"
	"os"
	"fmt"
	"net"
)

const expectedRequest = "Hi, I am client!"
const expectedReply = "Nice to meet you. I am server!"

var reply chan Message

func TestClientServer(t *testing.T) {
	reply = make(chan Message, 1)
	defer close(reply)

	const port uint = 9999

	server := startServer(t, port)
	defer server.Disconnect()

	client := startClient(t, port)
	defer client.Disconnect()

	msg := <-reply
	actualReply := string([]byte(msg))
	if actualReply != expectedReply {
		t.Fatalf("TestClientServer expected reply %q got %q.", expectedReply, actualReply)
	}
}

func TestPeer(t *testing.T) {
	reply = make(chan Message, 1)
	defer close(reply)

	peer0 := startPeer(t, 9999)
	defer peer0.Disconnect()
	peer0.AddHandler(receiveReply)

	peer1 := startPeer(t, 9988)
	defer peer1.Disconnect()
	peer1.AddHandler(sendReply)

	msg := []byte(expectedRequest)
	peer1Addr, err := net.ResolveUDPAddr(peer1.sock.LocalAddr().String())
	if err != nil {
		t.Fatalf("TestPeer could not resolve peer address: %s.", err)
	}
	peer0.SendTo(msg, peer1Addr)

	msg = <-reply
	actualReply := string([]byte(msg))
	if actualReply != expectedReply {
		t.Fatalf("TestPeer expected reply %q got %q.", expectedReply, actualReply)
	}
}

// Starts and returns a connector which listens to the specified port on localhost
func startPeer(t *testing.T, port uint) *Conn {
	conn := NewConn()
	go monitor(conn.Err, t)
	err := conn.Listen(port)
	if err != nil {
		t.Fatalf("Cannot start peer: %q", err)
		return nil
	}
	return conn
}

// Start a server which listens to the specified port on localhost
func startServer(t *testing.T, port uint) *Conn {
	conn := NewConn()
	go monitor(conn.Err, t)
	conn.AddHandler(sendReply)
	err := conn.Listen(port)
	if err != nil {
		t.Fatalf("Cannot start server: %q", err)
		return nil
	}
	return conn
}

// Start a client which sends packets to the specified port on localhost
func startClient(t *testing.T, port uint) *Conn {
	conn := NewConn()
	go monitor(conn.Err, t)
	conn.AddHandler(receiveReply)

	addr := fmt.Sprintf("127.0.0.1:%d",port)
	if err := conn.Dial(addr); err != nil {
		t.Fatalf("Cannot connect to server: %q", err)
		return nil
	}

	msg := []byte(expectedRequest)
	conn.Send(msg)

	return conn
}

// Fail all tests if connector encounters error.
// Since testing.T is thread-safe, this function can be run as a goroutine.
func monitor(errors <-chan os.Error, t *testing.T) {
	for err := range errors {
		t.Fatalf("Encountered unexpected error %q", err)
	}
}

// Event handler which responds to the sender of an incoming packet with the expected message.
func sendReply(conn *Conn, p *Packet) {
	actualRequest := string([]byte(p.Msg))
	if actualRequest == expectedRequest {
		conn.SendTo([]byte(expectedReply), p.Addr)
	} else {
		conn.SendTo([]byte("Go away!"), p.Addr)
	}
}

// Event handler to print the source address of an incoming packet.
func receiveReply(conn *Conn, p *Packet) {
	reply <- p.Msg
}
