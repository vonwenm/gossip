package transport

import (
	"testing"
	"os"
)

const expectedByte byte = 0x42

var reply chan Message

func TestConn(t *testing.T) {
	reply = make(chan Message, 1)

	startServer(t)
	startClient(t)

	msg := <-reply
	if len(msg) != 1 || msg[0] != expectedByte {
		t.Fatalf("Unexpected response message from server.")
	}
}

func startServer(t *testing.T) {
	conn := NewConn()
	go monitor(conn.Err, t)
	conn.AddHandler(sendReply)
	err := conn.Listen(9999)
	if err != nil {
		t.Fatalf("Cannot start server: %q", err)
		return
	}
}

func startClient(t *testing.T) {
	conn := NewConn()
	go monitor(conn.Err, t)
	conn.AddHandler(receiveReply)
	if err := conn.Dial("127.0.0.1:9999"); err != nil {
		t.Fatalf("Cannot connect to server: %q", err)
		return
	}

	msg := []byte("Hi, there!")
	conn.Send(msg)
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
	conn.SendTo([]byte{expectedByte}, p.Addr)
}

// Event handler to print the source address of an incoming packet.
func receiveReply(conn *Conn, p *Packet) {
	reply <- p.Msg
}
