package instrument

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOriginRoot(t *testing.T) {
	ipWithASN := "149.154.165.96"
	p := &defaultInstrument{
		ispLookup: &mockISPLookup{
			ASNS: map[string]string{
				ipWithASN: "AS62041",
			},
		},
	}
	requireSuccess := func(expected, input string) {
		actual, err := p.originRoot(input)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
	requireSuccess("facebook.com", "sub.facebook.com")
	requireSuccess("facebook.com", "facebook.com")
	requireSuccess("facebook", "facebook")
	requireSuccess("facebook.com", "157.240.221.48")
	requireSuccess("AS62041", ipWithASN) // Telegram IP addresses don't resolve, but we can get their ASN
}

type mockISPLookup struct {
	ASNS map[string]string
}

func (m *mockISPLookup) ISP(ip net.IP) string {
	return ""
}

func (m *mockISPLookup) ASN(ip net.IP) string {
	return m.ASNS[ip.String()]
}
