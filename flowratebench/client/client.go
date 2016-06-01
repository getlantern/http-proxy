package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"github.com/mxk/go-flowrate/flowrate"
)

var (
	addr        = flag.String("addr", "localhost:14082", "The address to which to send data")
	concurrency = flag.Int("concurrency", 10, "How many concurrent clients to run")
	limit       = flag.Int64("limit", 0, "Rate limit in bytes per second, set to 0 to not limit")

	data = make([]byte, 256)

	wg sync.WaitGroup
)

func main() {
	flag.Parse()

	log.Printf("Running %d clients agains %v", *concurrency, *addr)
	wg.Add(*concurrency)
	for i := 0; i < *concurrency; i++ {
		go client()
	}

	wg.Wait()
	log.Println("All clients finished")
}

func client() {
	defer wg.Done()
	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		log.Printf("Unable to dial %v: %v", *addr, err)
		return
	}

	var r io.Reader = conn
	var w io.Writer = conn

	if *limit > 0 {
		r = flowrate.NewReader(r, *limit)
		w = flowrate.NewWriter(w, *limit)
	}

	go func() {
		io.Copy(ioutil.Discard, r)
	}()

	for {
		w.Write(data)
	}
}
