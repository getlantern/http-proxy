// Add required headers to config-server requests and change the scheme to HTTPS.
// Ref https://github.com/getlantern/config-server/issues/4

package configserverfilter

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("configServerFilter")

type Options struct {
	AuthToken string
	Domains   []string
}

type ConfigServerFilter struct {
	*Options
}

func New(opts *Options) *ConfigServerFilter {
	if opts.AuthToken == "" || len(opts.Domains) == 0 {
		panic(errors.New("should set both config-server auth token and domains"))
	}
	log.Debugf("Will attach %s header on GET requests to %+v", common.CfgSvrAuthTokenHeader, opts.Domains)
	return &ConfigServerFilter{opts}
}

func (f *ConfigServerFilter) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	f.RewriteIfNecessary(req)
	return next()
}

func (f *ConfigServerFilter) RewriteIfNecessary(req *http.Request) {
	// It's unlikely that config-server will add non-GET public endpoint.
	// Bypass all other methods, especially CONNECT (https).
	if req.Method == "GET" && in(req.Host, f.Domains) {
		f.rewrite(req)
	}
}

func (f *ConfigServerFilter) rewrite(req *http.Request) {
	// use https between proxy and config-servers.
	req.URL.Scheme = "https"
	if !strings.HasSuffix(req.Host, ":443") {
		req.Host = strings.TrimSuffix(req.Host, ":80") + ":443"
	}
	req.Header.Set(common.CfgSvrAuthTokenHeader, f.AuthToken)
	log.Debugf("Attached %s header to \"GET %s\", host %s", common.CfgSvrAuthTokenHeader, req.URL.String(), req.Host)
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return
	}
	req.Header.Set(common.CfgSvrClientIPHeader, host)
	log.Debugf("Set %s as %s to \"GET %s\"", common.CfgSvrClientIPHeader, host, req.URL.String())
}

func in(host string, domains []string) bool {
	domain, _, err := net.SplitHostPort(host)
	if err != nil {
		domain = host
	}
	for _, d := range domains {
		if domain == d {
			return true
		}
	}
	return false
}
