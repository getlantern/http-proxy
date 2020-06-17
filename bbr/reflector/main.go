// reflector is a simple HTTP server that sends random data to a client and
// transmits the BBR information in an HTTP trailer.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getlantern/bbrconn"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/v2/common"
)

var (
	log = golog.LoggerFor("bbr.reflector")

	addr = flag.String("addr", ":80", "address to listen for HTTP connections, defaults to port 80 on all interfaces")

	// data is 1 KB of random data
	data = common.RandStringData(1024)
)

func main() {
	flag.Parse()

	err := http.ListenAndServe(*addr, http.HandlerFunc(handle))
	if err != nil {
		log.Fatal(err)
	}
}

func handle(resp http.ResponseWriter, req *http.Request) {
	_size := req.URL.Path[1:]
	size, err := strconv.Atoi(_size)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "Path could not be parsed to integer size: %v\n", size)
		return
	}

	for i := 0; i < size; i++ {
		resp.Write(data)
	}

	conn, buf, _ := resp.(http.Hijacker).Hijack()
	bconn, err := bbrconn.Wrap(conn, nil)
	if err != nil {
		log.Errorf("Error wrapping connection with BBR: %v", err)
		return
	}
	binfo, err := bconn.BBRInfo()
	if err != nil {
		log.Errorf("Unable to obtain BBR information: %v", err)
		return
	}

	estABE := float64(binfo.MaxBW) * 8 / 1000 / 1000
	log.Debugf("Reporting ABE %v", estABE)
	trailers := http.Header{}
	trailers.Set(common.BBRAvailableBandwidthEstimateHeader, fmt.Sprint(estABE))

	buf.WriteString("0\r\n") // eof
	trailers.Write(buf)

	buf.WriteString("\r\n") // end of trailers
	buf.Flush()
	conn.Close()
}
