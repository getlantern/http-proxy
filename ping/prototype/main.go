// This is just a small prototype program that runs the ping monitoring so we
// can observe the results on an arbitrary machine without having to run the
// full http-proxy.
package main

import (
	"github.com/getlantern/http-proxy-lantern/ping"
	"time"
)

func main() {
	ping.New(nil)
	time.Sleep(168 * time.Hour)
}
