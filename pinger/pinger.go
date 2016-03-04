package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy-lantern/common"
)

var (
	log = golog.LoggerFor("pinger")
)

func main() {
	proxy := flag.String("proxy", "", "The server to hit")
	token := flag.String("token", "", "The token of the server to hit")
	pause := flag.Int64("pause", 30, "Pause time in seconds")

	flag.Parse()

	// Note - this request will never actually go to Google, we just need a
	// valid URL.
	url := fmt.Sprintf("https://www.google.com/humans.txt")
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Dial: func(network, addr string) (net.Conn, error) {
				// Always dial the proxy
				start := time.Now()
				conn, err := net.Dial("tcp", *proxy)
				delta := time.Now().Sub(start)
				log.Debugf("Dial time: %v", delta)
				return conn, err
			},
		},
	}

	for {
		makeRequest(client, url, *token, "small")
		makeRequest(client, url, *token, "medium")
		makeRequest(client, url, *token, "large")
		time.Sleep(time.Duration(*pause) * time.Second)
	}
}

func makeRequest(client *http.Client, url, token, size string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Unable to create request: %v", err)
	}
	req.Header.Set(common.DeviceIdHeader, "9999")
	req.Header.Set(common.TokenHeader, token)
	req.Header.Set(common.PingHeader, size)
	start := time.Now()
	resp, err := client.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Errorf("Unable to issue request: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		log.Errorf("Bad response status: %v", resp.StatusCode)
		if resp.Body != nil {
			io.Copy(os.Stderr, resp.Body)
		}
	} else {
		io.Copy(ioutil.Discard, resp.Body)
		delta := time.Now().Sub(start)
		log.Debugf("%v Finished %v request in %v", time.Now(), size, delta)
	}
}
