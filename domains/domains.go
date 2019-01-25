package domains

import (
	"net"
	"net/http"
	"strings"
)

// Config represents the configuration for a given domain
type Config struct {
	// Domain contains the name of the domain
	Domain string

	// Unthrottled indicates that this domain should not be subject to throttling.
	Unthrottled bool

	// RewriteToHTTPS indicates that HTTP requests to this domain should be
	// rewritten to HTTPS.
	RewriteToHTTPS bool

	// AddConfigServerHeaders indicates that we should add config server auth
	// tokens and client IP headers on requests to this domain
	AddConfigServerHeaders bool

	// AddForwardedFor indicates that we should include an X-Forwarded-For header
	// with the client's IP.
	AddForwardedFor bool

	// PassInternalHeaders indicates that headers starting with X-Lantern-* should
	// be passed to this domain.
	PassInternalHeaders bool
}

func (cfg *Config) WithRewriteToHTTPS() *Config {
	var cfg2 = *cfg
	cfg2.RewriteToHTTPS = true
	return &cfg2
}

func (cfg *Config) WithAddConfigServerHeaders() *Config {
	var cfg2 = *cfg
	cfg2.AddConfigServerHeaders = true
	return &cfg2
}

var (
	internal = &Config{
		Unthrottled:         true,
		AddForwardedFor:     true,
		PassInternalHeaders: true,
	}

	externalUnthrottled = &Config{
		Unthrottled: true,
	}
)

var configs = map[string]*Config{
	"config.getiantem.org":                     internal.WithRewriteToHTTPS().WithAddConfigServerHeaders(),
	"config-staging.getiantem.org":             internal.WithRewriteToHTTPS().WithAddConfigServerHeaders(),
	"api.getiantem.org":                        internal.WithRewriteToHTTPS(),
	"api-staging.getiantem.org":                internal.WithRewriteToHTTPS(),
	"getlantern.org":                           internal,
	"lantern.io":                               internal,
	"innovatelabs.io":                          internal,
	"getiantem.org":                            internal,
	"lantern-pro-server.herokuapp.com":         internal,
	"lantern-pro-server-staging.herokuapp.com": internal,
	"adyenpayments.com":                        externalUnthrottled,
	"adyen.com":                                externalUnthrottled,
	"stripe.com":                               externalUnthrottled,
	"paymentwall.com":                          externalUnthrottled,
	"alipay.com":                               externalUnthrottled,
	"app-measurement.com":                      externalUnthrottled,
	"fastworldpay.com":                         externalUnthrottled,
	"firebaseremoteconfig.googleapis.com":      externalUnthrottled,
	"firebaseio.com":                           externalUnthrottled,
	"optimizely.com":                           externalUnthrottled,
}

// ConfigForRequest returns a config that is the superset of all permissions for
// domains matching the req.Host.
func ConfigForRequest(req *http.Request) *Config {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		host = req.Host
	}
	return ConfigForHost(host)
}

// ConfigForHost returns a config that is the superset of all permissions for
// domains matching the host.
func ConfigForHost(host string) *Config {
	cfg := &Config{Domain: host}

	for d, dcfg := range configs {
		if host == d || strings.HasSuffix(host, "."+d) {
			cfg.Unthrottled = cfg.Unthrottled || dcfg.Unthrottled
			cfg.RewriteToHTTPS = cfg.RewriteToHTTPS || dcfg.RewriteToHTTPS
			cfg.AddConfigServerHeaders = cfg.AddConfigServerHeaders || dcfg.AddConfigServerHeaders
			cfg.AddForwardedFor = cfg.AddForwardedFor || dcfg.AddForwardedFor
			cfg.PassInternalHeaders = cfg.PassInternalHeaders || dcfg.PassInternalHeaders
		}
	}

	return cfg
}
