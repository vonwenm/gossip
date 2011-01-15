package gossip

import (
	"testing"
	"os"
	"net"
	"strconv"
)

func TestBroadcast(t *testing.T) {
	requester := openSocket(t, 8100)
	responder := openSocket(t, 8200)

	c := make(chan bool)
	go respond(t, responder)
	go receive(t, requester, c)
	addr := &net.UDPAddr{IP:net.IPv4bcast, Port:8200}
	request := []byte("Some request")
	_, err := requester.WriteTo(request, addr)
	if err != nil {
		t.Fatalf("Unable to respond to request: %s", err)
	}

	<-c
}

// Strobes the channel when the connection encounters an incoming packet.
func receive(t *testing.T, conn *net.UDPConn, c chan bool) {
	buff := make([]byte, 16)
	_, _, err := conn.ReadFromUDP(buff)
	if err != nil {
		t.Fatalf("Cannot serve request: %s", err)
	}
	
	c <- true
}


// Listens for an incoming packet and replies to it.
func respond(t *testing.T, conn *net.UDPConn) {
	buff := make([]byte, 16)
	_, addr, err := conn.ReadFromUDP(buff)
	if err != nil {
		t.Fatalf("Cannot serve request: %s", err)
	}
	
	response := []byte("Some reply")
	if _, err = conn.WriteTo(response, addr); err != nil {
		t.Fatalf("Cannot write response: %s", err)
	}
}

// Open socket which can listen to any local address on the specified port.
func openSocket(t *testing.T, port uint) *net.UDPConn {
	var err os.Error
	var addr *net.UDPAddr
	var conn *net.UDPConn

	addr, err = net.ResolveUDPAddr(":" + strconv.Uitoa(port))
	if err != nil {
		t.Fatalf("Cannot resolve address: %s", err)
	}
	conn, err = net.ListenUDP("udp4", addr)
	if err != nil {
		t.Fatalf("Cannot listen to address %s: %s", addr, err)
	}

	return conn
}
