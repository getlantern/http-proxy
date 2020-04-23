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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestWrapConnErrorHandler(t *testing.T) {
	var wg sync.WaitGroup
	f := NewPrometheus(geo.NoLookup{}, CommonLabels{}).WrapConnErrorHandler("test", func(conn net.Conn, err error) {
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
	result, err := prometheus.DefaultRegisterer.(*prometheus.Registry).Gather()
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
