package domains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalFull(t *testing.T) {
	cfg := ConfigForHost("config.getiantem.org")
	assert.True(t, cfg.Unthrottled)
	assert.True(t, cfg.RewriteToHTTPS)
	assert.True(t, cfg.AddConfigServerHeaders)
	assert.True(t, cfg.AddForwardedFor)
	assert.True(t, cfg.PassInternalHeaders)
	assert.Equal(t, "config.getiantem.org", cfg.Domain)
}

func TestInternalRewriteToHTTPS(t *testing.T) {
	cfg := ConfigForHost("api.getiantem.org")
	assert.True(t, cfg.Unthrottled)
	assert.True(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.True(t, cfg.AddForwardedFor)
	assert.True(t, cfg.PassInternalHeaders)
	assert.Equal(t, "api.getiantem.org", cfg.Domain)
}

func TestInternal(t *testing.T) {
	cfg := ConfigForHost("other.getiantem.org")
	assert.True(t, cfg.Unthrottled)
	assert.False(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.True(t, cfg.AddForwardedFor)
	assert.True(t, cfg.PassInternalHeaders)
	assert.Equal(t, "other.getiantem.org", cfg.Domain)
}

func TestExternalUnthrottled(t *testing.T) {
	cfg := ConfigForHost("alipay.com")
	assert.True(t, cfg.Unthrottled)
	assert.False(t, cfg.RewriteToHTTPS)
	assert.False(t, cfg.AddConfigServerHeaders)
	assert.False(t, cfg.AddForwardedFor)
	assert.False(t, cfg.PassInternalHeaders)
	assert.Equal(t, "alipay.com", cfg.Domain)
}
