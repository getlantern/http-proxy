package domains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalFull(t *testing.T) {
	cfg := ConfigForHost("sub.config.getiantem.org")
	assert.True(t, cfg.Unthrottled)
	assert.True(t, cfg.RewriteToHTTPS)
	assert.True(t, cfg.AddConfigServerHeaders)
	assert.True(t, cfg.AddForwardedFor)
	assert.True(t, cfg.PassInternalHeaders)
	assert.Equal(t, "sub.config.getiantem.org", cfg.Host)
}

func TestInternalRewriteToHTTPS(t *testing.T) {
	cfg := ConfigForHost("sub.api.getiantem.org")
	assert.True(t, cfg.Unthrottled)
	assert.True(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.True(t, cfg.AddForwardedFor)
	assert.True(t, cfg.PassInternalHeaders)
	assert.Equal(t, "sub.api.getiantem.org", cfg.Host)
}

func TestInternal(t *testing.T) {
	cfg := ConfigForHost("other.getiantem.org")
	assert.True(t, cfg.Unthrottled)
	assert.False(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.True(t, cfg.AddForwardedFor)
	assert.True(t, cfg.PassInternalHeaders)
	assert.Equal(t, "other.getiantem.org", cfg.Host)
}

func TestExternalUnthrottled(t *testing.T) {
	cfg := ConfigForHost("sub.alipay.com")
	assert.True(t, cfg.Unthrottled)
	assert.False(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.False(t, cfg.AddForwardedFor)
	assert.False(t, cfg.PassInternalHeaders)
	assert.Equal(t, "sub.alipay.com", cfg.Host)
}

func TestUnknown(t *testing.T) {
	cfg := ConfigForHost("unknown.com")
	assert.False(t, cfg.Unthrottled)
	assert.False(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.False(t, cfg.AddForwardedFor)
	assert.False(t, cfg.PassInternalHeaders)
	assert.Equal(t, "unknown.com", cfg.Host)
}
