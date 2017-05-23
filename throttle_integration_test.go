package proxy

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/testredis"
	"github.com/getlantern/http-proxy-lantern/throttle"
	. "github.com/getlantern/waitforserver"
	"github.com/stretchr/testify/assert"
)

const (
	freeServerAddr = "127.0.0.1:18711"
	proServerAddr  = "127.0.0.1:18712"
)

func TestThrottlingFree(t *testing.T) {
	doTestThrottling(t, false, freeServerAddr)
}

func TestThrottlingPro(t *testing.T) {
	doTestThrottling(t, true, proServerAddr)
}

func doTestThrottling(t *testing.T, pro bool, serverAddr string) {
	origMeasuredReportingInterval := measuredReportingInterval
	measuredReportingInterval = 10 * time.Millisecond
	defer func() {
		measuredReportingInterval = origMeasuredReportingInterval
	}()

	r, err := testredis.Open()
	if !assert.NoError(t, err) {
		return
	}
	defer r.Close()

	rc := r.Client()
	defer rc.Close()

	if !assert.NoError(t, rc.HMSet("_throttle:desktop", throttle.DefaultCountryCode, "10485760|1048576").Err()) {
		return
	}
	if !assert.NoError(t, rc.HMSet("_throttle:mobile", throttle.DefaultCountryCode, "10485760|1048576").Err()) {
		return
	}

	proxy := &Proxy{
		Addr:               serverAddr,
		ReportingRedisAddr: "redis://" + r.Addr(),
		Token:              validToken,
		EnableReports:      true,
		IdleTimeout:        1 * time.Minute,
		Pro:                pro,
	}
	go func() {
		assert.NoError(t, proxy.ListenAndServe())
	}()

	if !assert.NoError(t, WaitForServer("tcp", serverAddr, 10*time.Second)) {
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://" + serverAddr)
			},
		},
	}

	makeRequest := func(url string) (*http.Response, error) {
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set(common.TokenHeader, validToken)
		req.Header.Set(common.DeviceIdHeader, deviceId)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(ioutil.Discard, resp.Body)
		return resp, err
	}

	resp, err := makeRequest("http://download.thinkbroadband.com/5MB.zip")
	if !assert.NoError(t, err) {
		return
	}

	time.Sleep(measuredReportingInterval * 30)
	resp, err = makeRequest("http://www.scmp.com/frontpage/international")
	if !assert.NoError(t, err) {
		return
	}

	xbq := resp.Header.Get(common.XBQ)
	if pro {
		assert.Empty(t, xbq)
	} else {
		if !assert.NotEmpty(t, xbq) {
			return
		}

		parts := strings.Split(xbq, "/")
		if !assert.Len(t, parts, 3) {
			return
		}

		assert.NotEqual(t, "0", parts[0], "Should show some usage")
		assert.Equal(t, "10", parts[1], "Should show correct bandwidth limit")
	}

	result, err := rc.HMGet("_client:"+deviceId, "bytesIn", "bytesOut", "countryCode", "clientIP").Result()
	if !assert.NoError(t, err) {
		return
	}

	bytesIn, err := strconv.Atoi(result[0].(string))
	if !assert.NoError(t, err) {
		return
	}
	bytesOut, err := strconv.Atoi(result[1].(string))
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, bytesIn > 0)
	assert.True(t, bytesOut > 0)
	assert.Equal(t, "", result[2])
	assert.Equal(t, "127.0.0.1", result[3])
}
