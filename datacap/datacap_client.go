package datacap

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/getlantern/errors"
)

type DatacapSidecarClient interface {
	TrackDatacapUsage(ctx context.Context, deviceID string, bytesUsed int64, countryCode, platform string) (usage *TrackDatacapResponse, err error)

	GetDatacapUsage(ctx context.Context, deviceID string) (usage *TrackDatacapResponse, err error)
}

type Config struct {
	SidecarAddr string
	HTTPClient  *http.Client
}

type datacapClient struct {
	config Config
}

// TrackDatacapRequest represents the request to track data usage
type TrackDatacapRequest struct {
	DeviceID    string `json:"deviceId"`
	BytesUsed   int64  `json:"bytesUsed"`
	CountryCode string `json:"countryCode"`
	Platform    string `json:"platform"`
}

// TrackDatacapResponse represents the response from tracking data usage
type TrackDatacapResponse struct {
	Allowed        bool  `json:"allowed"`
	RemainingBytes int64 `json:"remainingBytes"`
	CapLimit       int64 `json:"capLimit"`
	ExpiryTime     int64 `json:"expiryTime"`
}

func NewClient(config Config) DatacapSidecarClient {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &datacapClient{
		config: config,
	}
}

func (c *datacapClient) TrackDatacapUsage(ctx context.Context, deviceID string, bytesUsed int64, countryCode, platform string) (*TrackDatacapResponse, error) {
	req := TrackDatacapRequest{
		DeviceID:    deviceID,
		BytesUsed:   bytesUsed,
		CountryCode: countryCode,
		Platform:    platform,
	}

	// Ensure the sidecar address has a trailing slash
	sidecarAddr := c.config.SidecarAddr
	if !strings.HasSuffix(sidecarAddr, "/") {
		sidecarAddr += "/"
	}

	url := sidecarAddr + "data-cap/usage"

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, errors.New("failed to marshal request for tracking datacap usage: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, errors.New("failed to create request for tracking datacap usage: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, errors.New("failed to send request to sidecar: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("sidecar returned non-200 status: %d", resp.StatusCode)
	}

	var trackResp TrackDatacapResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackResp); err != nil {
		return nil, errors.New("failed to decode response for tracking datacap usage: %v", err)
	}

	log.Debugf("Track response for device %s: allowed=%v, remaining=%d",
		deviceID, trackResp.Allowed, trackResp.RemainingBytes)

	return &trackResp, nil
}

func (c *datacapClient) GetDatacapUsage(ctx context.Context, deviceID string) (*TrackDatacapResponse, error) {
	// Ensure the sidecar address has a trailing slash
	sidecarAddr := c.config.SidecarAddr
	if !strings.HasSuffix(sidecarAddr, "/") {
		sidecarAddr += "/"
	}

	url := sidecarAddr + "data-cap/device/" + deviceID

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.New("failed to create request for tracking datacap usage: %v", err)
	}

	resp, err := c.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, errors.New("failed to send request to sidecar: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("sidecar returned non-200 status: %d", resp.StatusCode)
	}

	var trackResp TrackDatacapResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackResp); err != nil {
		return nil, errors.New("failed to decode response for tracking datacap usage: %v", err)
	}

	log.Debugf("Track response for device %s: allowed=%v, remaining=%d",
		deviceID, trackResp.Allowed, trackResp.RemainingBytes)

	return &trackResp, nil
}
