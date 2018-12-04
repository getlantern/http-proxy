// +build !linux

package pcap

// Capture doesn't do anything
func Capture() func(string) {
	return func(ip string) {}
}
