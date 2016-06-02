package main

import (
	"flag"
	"io"
	"log"
	"net"

	"github.com/mxk/go-flowrate/flowrate"
)

var (
	addr  = flag.String("addr", ":14082", "The address at which to listen")
	limit = flag.Int64("limit", 0, "Rate limit in bytes per second, set to 0 to not limit")
)

func main() {
	flag.Parse()

	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Unable to listen at %v: %v", *addr, err)
	}

	log.Printf("Listening at: %v", l.Addr())
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Unable to accept connection: %v", err)
		}
		go echo(conn)
	}
}

func echo(conn net.Conn) {
	var r io.Reader = conn
	var w io.Writer = conn

	if *limit > 0 {
		r = flowrate.NewReader(r, *limit)
		w = flowrate.NewWriter(w, *limit)
	}

	io.Copy(w, r)
}
