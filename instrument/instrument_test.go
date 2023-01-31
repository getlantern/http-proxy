package instrument

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/geo"
	"github.com/getlantern/mockconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapConnErrorHandler(t *testing.T) {
	var wg sync.WaitGroup
	instrument := NewPrometheus(geo.NoLookup{}, geo.NoLookup{}, CommonLabels{})
	f := instrument.WrapConnErrorHandler("test", func(conn net.Conn, err error) {
		time.Sleep(100 * time.Millisecond)
		wg.Done()
	})
	var buf bytes.Buffer
	response := bytes.NewReader([]byte{0, 0, 0})
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go f(mockconn.New(&buf, response), errors.New("abc"))
	}
	wg.Wait()
	result, err := instrument.registry.Gather()
	assert.NoError(t, err)
	var errors, consec_errors float64
	for _, metric := range result {
		switch *metric.Name {
		case "test_consec_per_client_ip_errors_total":
			consec_errors = *metric.Metric[0].Counter.Value
		case "test_errors_total":
			errors = *metric.Metric[0].Counter.Value
		}
	}
	assert.Equal(t, 5.0, errors)
	assert.Equal(t, 1.0, consec_errors)
}

func TestOriginRoot(t *testing.T) {
	ipWithASN := "149.154.165.96"
	p := &PromInstrument{
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
