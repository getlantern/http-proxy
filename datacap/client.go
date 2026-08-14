// Package datacap reports per-device byte usage to the local datacap sidecar
// and enforces the throttle verdict the sidecar returns.
//
// It replaces the reporting-Redis path (see the redis package): instead of
// writing `_client:<deviceID>` hashes to a shared Redis and reading cohort
// settings back out of a `_throttle` key, the proxy POSTs deltas to a sidecar
// on localhost that owns the cap accounting and answers with the current
// throttle state. The wire contract is the same one lantern-box speaks
// (getlantern/lantern-box tracker/datacap), so a device's traffic accumulates
// into one counter no matter which proxy flavor carried it.
package datacap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultHTTPTimeout bounds a single report round-trip. The sidecar is on
// localhost and answers from memory, so anything slower than this means it is
// wedged and the delta is better retried on the next cycle than left in flight.
const DefaultHTTPTimeout = 10 * time.Second

// Report is the body of POST /data-cap/. BytesUsed is a delta since the last
// report, not a running total — the sidecar accumulates.
type Report struct {
	DeviceID    string `json:"deviceId"`
	CountryCode string `json:"countryCode"`
	Platform    string `json:"platform"`
	BytesUsed   int64  `json:"bytesUsed"`
}

// Status is the sidecar's answer: the throttle verdict plus enough of the
// device's cap state to render the XBQ headers clients show in their usage UI.
type Status struct {
	Throttle   bool  `json:"throttle"`
	CapLimit   int64 `json:"capLimit"`
	ExpiryTime int64 `json:"expiryTime"` // Unix seconds
	BytesUsed  int64 `json:"bytesUsed"`
}

// Client talks to the datacap sidecar over HTTP.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient returns a Client posting to baseURL, e.g. "http://127.0.0.1:8078".
// A bare host:port is assumed to be plain HTTP: the sidecar listens on loopback
// (or the phost bridge address) without TLS.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			// A bare Transport also deliberately ignores HTTP_PROXY et al. —
			// this client only ever talks to the local sidecar. The idle pool
			// matches the flush fan-out so a full cycle's connections are all
			// reusable instead of the excess being closed each cycle.
			Transport: &http.Transport{
				MaxIdleConnsPerHost: flushConcurrency,
			},
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// ReportUsage posts a usage delta and returns the device's updated cap state.
func (c *Client) ReportUsage(ctx context.Context, report *Report) (*Status, error) {
	body, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/data-cap/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("sidecar returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode status: %w", err)
	}
	return &status, nil
}
