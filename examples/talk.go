// Simple chat over UDP protocol
package main

import (
	"flag"
	"fmt"
	"os"
	"bufio"
	"net"
	"github.com/ahorn/gossip"
)

var port *uint = flag.Uint("p", 9999, "listening port")

func usage() {
	report("usage: talk [flags] addr")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}

	conn := gossip.NewConn()
	go monitor(conn.Err)

	conn.AddHandler(echo)
	if err := conn.Listen(*port); err != nil {
		report(fmt.Sprintf("Cannot listen on port %s because %s", *port, err))
		os.Exit(3)
	}
	defer conn.Disconnect()

	// parse the destination address of the form host:port
	addr := flag.Arg(0)
	udpAddr, err := net.ResolveUDPAddr(addr)
	if err != nil {
		report(fmt.Sprintf("Cannot resolve %q because %s", addr, err))
		os.Exit(3)
	}

	lines := make(chan string, 8)
	go read(lines)

	// loop until EOF
	for line := range lines {
		conn.SendTo([]byte(line), udpAddr)
	}
}

// Reads lines from the text console and pipe them to the 'lines' channel.
// Once EOF is encountered, the channel is automatically closed.
func read(lines chan<- string) {
	reader := bufio.NewReader(os.Stdin)

	var line string
	var err os.Error
	for {
		line, err = reader.ReadString('\n')
		if err == os.EOF {
			break
		}
		lines <- line
	}

	close(lines)
}

// Event handler to print the response of the conversation partner
func echo(conn *gossip.Conn, p *gossip.Packet) {
	line := string([]byte(p.Msg))
	fmt.Print(">", line)
}

// Report socket errors
func monitor(errors <-chan os.Error) {
	for err := range errors {
		report(err)
	}
}

// Print an error
func report(err interface{}) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
}
