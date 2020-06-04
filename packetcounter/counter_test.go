package packetcounter

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name        string
	addrs       []string
	result, err string
}

func TestComposeBPF(t *testing.T) {
	testCases := []testCase{
		testCase{"nil addrs", nil, "", "no address is configured on interface"},
		testCase{"empty addrs", []string{}, "", "no address is configured on interface"},
		testCase{"ipv4 addr", []string{"127.0.0.1"}, "tcp and src port 443 and src host 127.0.0.1", ""},
		testCase{"ipv6 addr", []string{"2001:db8::68"}, "tcp and src port 443 and src host 2001:db8::68", ""},
		testCase{"multiple addrs", []string{"127.0.0.1", "2001:db8::68"}, "tcp and src port 443 and (src host 127.0.0.1 or src host 2001:db8::68)", ""},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var addrs []net.IP
			if tc.addrs == nil {
				addrs = nil
			} else {
				for _, addr := range tc.addrs {
					addrs = append(addrs, net.ParseIP(addr))
				}
			}
			result, err := composeBPF(addrs, "443")
			assert.Equal(t, tc.result, result)
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				if assert.Error(t, err) {
					assert.Equal(t, tc.err, err.Error())
				}
			}
		})
	}
}
