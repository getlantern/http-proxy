package proxy

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/getlantern/golog/testlog"
	"github.com/getlantern/testredis"
	. "github.com/getlantern/waitforserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/throttle"
)

func TestThrottling(t *testing.T) {
	stopCapture := testlog.Capture(t)
	defer stopCapture()

	origMeasuredReportingInterval := measuredReportingInterval
	measuredReportingInterval = 10 * time.Millisecond
	defer func() {
		measuredReportingInterval = origMeasuredReportingInterval
	}()

	r, err := testredis.OpenUnstarted()
	require.NoError(t, err)
	defer r.Close()

	throttleThreshold := 10485760
	throttleRate := 1024
	t.Run("free_config_when_redis_is_down", func(t *testing.T) {
		doTestThrottling(t, false, "127.0.0.1:18707", false, throttleThreshold, throttleRate, r)
	})

	r.Start()

	t.Run("disabling_throttling_via_redis", func(t *testing.T) {
		doTestThrottling(t, true, "127.0.0.1:18709", true, 0, throttleRate, r)
	})

	t.Run("free_config_from_redis", func(t *testing.T) {
		doTestThrottling(t, false, "127.0.0.1:18711", true, throttleThreshold, throttleRate, r)
	})
}

func doTestThrottling(t *testing.T, pro bool, serverAddr string, redisIsUp bool, throttleThreshold, throttleRate int, r testredis.Redis) {
	deviceId := fmt.Sprintf("dev-%d", rand.Int())
	sizeHeader := "X-Test-Size"
	originSite := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		n, _ := strconv.Atoi(req.Header.Get(sizeHeader))
		io.CopyN(rw, rand.New(rand.NewSource(time.Now().UnixNano())), int64(n))
	}))
	originAddr := originSite.Listener.Addr().String()
	log.Debugf("Waiting for origin server at %s...", originAddr)
	require.NoError(t, WaitForServer("tcp", originAddr, 10*time.Second))

	rc := r.Client()
	defer rc.Close()

	if redisIsUp {
		settings := fmt.Sprintf(`{"default": { "default": [{"capResets": "monthly", "threshold": %d, "rate": %d}] } }`, throttleThreshold, throttleRate)
		require.NoError(t, rc.Set("_throttle", settings, 0).Err())
	}

	durationForBytes := func(bytes, readers int) time.Duration {
		// the buckets will be full initially, so these will be available immediately.
		lbytes := bytes - (readers * throttleRate)
		if lbytes <= 0 {
			return 0
		}

		return time.Duration(1000*float64(lbytes)/float64(readers*throttleRate)) * time.Millisecond
	}

	proxy := &Proxy{
		HTTPAddr:                serverAddr,
		ReportingRedisAddr:      "redis://" + r.Addr(),
		Token:                   validToken,
		EnableReports:           true,
		IdleTimeout:             1 * time.Minute,
		Pro:                     pro,
		ThrottleRefreshInterval: throttle.DefaultRefreshInterval,
		TestingLocal:            true,
		GoogleSearchRegex:       "bequiet",
		GoogleCaptchaRegex:      "bequiet",
	}
	go func() {
		assert.NoError(t, proxy.ListenAndServe())
	}()

	require.NoError(t, WaitForServer("tcp", serverAddr, 10*time.Second))

	makeRequest := func(u string, testSize int) (*http.Response, int, error) {
		var conn *ReadSizeConn
		client := &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
				Proxy: func(req *http.Request) (*url.URL, error) {
					return url.Parse("http://" + serverAddr)
				},
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					var d net.Dialer
					c, err := d.DialContext(ctx, network, addr)
					if err != nil {
						return nil, err
					}
					wrapped := &ReadSizeConn{Conn: c}
					conn = wrapped
					return wrapped, nil
				},
			},
		}

		req, _ := http.NewRequest(http.MethodGet, u, nil)
		req.Header.Set(common.TokenHeader, validToken)
		req.Header.Set(common.DeviceIdHeader, deviceId)
		req.Header.Set(sizeHeader, strconv.Itoa(testSize))

		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}

		_, err = io.Copy(ioutil.Discard, resp.Body)

		rs := 0
		if conn != nil {
			rs = conn.readSize
		}
		return resp, rs, err
	}

	resp, _, err := makeRequest(originSite.URL, 9*1024*1024)
	require.NoError(t, err)

	resp, _, err = makeRequest(originSite.URL, 1024*1024)
	require.NoError(t, err)

	time.Sleep(time.Second)

	start := time.Now()
	resp, sz, err := makeRequest(originSite.URL, 3*throttleRate)
	require.NoError(t, err)
	xbq := resp.Header.Get(common.XBQHeader)
	xbqv2 := resp.Header.Get(common.XBQHeaderv2)
	if !redisIsUp || throttleThreshold <= 0 {
		assert.Empty(t, xbq)
		assert.Empty(t, xbqv2)
		return
	}

	if pro {
		assert.Empty(t, xbq)
	} else {
		assert.InDelta(t, durationForBytes(sz, 1), time.Since(start), float64(100*time.Millisecond),
			fmt.Sprintf("per connection throttling should be in effect for Free proxy sz=%d", sz))

		require.NotEmpty(t, xbq)

		parts := strings.Split(xbqv2, "/")
		require.Len(t, parts, 4)
		require.Len(t, strings.Split(xbq, "/"), 3)

		log.Debugf("XBQ is: %v", xbq)
		assert.NotEqual(t, "0", parts[0], "Should show some usage")
		assert.Equal(t, "10", parts[1], "Should show correct bandwidth limit")

		time.Sleep(time.Second)
		// Now test throttling concurrent connections from a single device
		readers := 16
		readSize := 3 * throttleRate

		errors := make(chan error, readers)
		var sz int64
		var wg sync.WaitGroup

		start := time.Now()
		for i := 0; i < readers; i++ {
			wg.Add(1)
			go func() {
				_, ss, err := makeRequest(originSite.URL, readSize)
				atomic.AddInt64(&sz, int64(ss))
				errors <- err
				wg.Done()
			}()
		}
		wg.Wait()
		endTime := time.Since(start)
		close(errors)
		for err := range errors {
			assert.NoError(t, err)
		}
		assert.InDelta(t, durationForBytes(int(sz), readers), endTime, float64(150*time.Millisecond),
			fmt.Sprintf("throttling should be applied to each connection generated by the device sz=%d", sz))
	}

	result, err := rc.HMGet("_client:"+deviceId, "bytesIn", "bytesOut", "countryCode", "clientIP").Result()
	require.NoError(t, err)

	bytesIn, err := strconv.Atoi(result[0].(string))
	require.NoError(t, err)
	bytesOut, err := strconv.Atoi(result[1].(string))
	require.NoError(t, err)
	assert.True(t, bytesIn > 0)
	assert.True(t, bytesOut > 0)
	assert.Equal(t, "", result[2])
	assert.Equal(t, "127.0.0.1", result[3])

	ttl, err := rc.TTL("_client:" + deviceId).Result()
	require.NoError(t, err)
	log.Debug(ttl)
}

// utility for observing bytes read
type ReadSizeConn struct {
	net.Conn
	readSize int
}

func (c *ReadSizeConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	c.readSize += n
	return
}
