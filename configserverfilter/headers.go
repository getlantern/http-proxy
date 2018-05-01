// Add required headers to config-server requests and change the scheme to HTTPS.
// Ref https://github.com/getlantern/config-server/issues/4

package configserverfilter

import (
	"errors"
	"net"
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("configServerFilter")

type Options struct {
	AuthToken string
	Domains   []string
}

type ConfigServerFilter struct {
	opts *Options
}

func New(opts *Options) *ConfigServerFilter {
	if opts.AuthToken == "" || len(opts.Domains) == 0 {
		panic(errors.New("should set both config-server auth token and domains"))
	}
	log.Debugf("Will attach %s header on GET requests to %+v", common.CfgSvrAuthTokenHeader, opts.Domains)

	return &ConfigServerFilter{
		opts: opts,
	}
}

func (f *ConfigServerFilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	f.RewriteIfNecessary(req)
	return next(ctx, req)
}

func (f *ConfigServerFilter) RewriteIfNecessary(req *http.Request) {
	matched := f.matchingDomains(req)
	if matched != "" {
		f.rewrite(matched, req)
	}
}

func (f *ConfigServerFilter) matchingDomains(req *http.Request) string {
	// It's unlikely that config-server will add non-GET public endpoint.
	// Bypass all other methods, especially CONNECT (https).
	if req.Method == "GET" {
		if matched := in(req.Host, f.opts.Domains); matched != "" {
			return matched
		}
	}
	return ""
}

func (f *ConfigServerFilter) rewrite(host string, req *http.Request) {
	req.URL.Scheme = "https"
	prevHost := req.Host
	req.Host = host + ":443"
	req.Header.Set(common.CfgSvrAuthTokenHeader, f.opts.AuthToken)
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return
	}
	req.Header.Set(common.CfgSvrClientIPHeader, ip)
	log.Debugf("Rewrote request from %s to %s as \"GET %s\", host %s", ip, prevHost, req.URL.String(), req.Host)
}

// in returns the host portion if it's is in the domains list, or returns ""
func in(hostport string, domains []string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	for _, d := range domains {
		if host == d {
			return d
		}
	}
	return ""
}
