// loader generates CPU load
package main

import (
	"flag"
	"time"
)

var (
	pause = flag.Int64("pause", 50000, "Pause time in nanoseconds. Make smaller to increase load.")
)

func main() {
	p := time.Duration(*pause)
	for {
		if p > 0 {
			time.Sleep(p)
		}
	}
}
