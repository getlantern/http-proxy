// Add required headers to config-server requests.
// Ref https://github.com/getlantern/config-server/issues/4

package configserverfilter

import (
	"errors"
	"net"
	"net/http"

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
	f.AttachHeaderIfNecessary(req)
	return next()
}

func (f *ConfigServerFilter) AttachHeaderIfNecessary(req *http.Request) {
	// It's unlikely that config-server will add non-GET public endpoint.
	// Bypass all other methods, especially CONNECT (https).
	if req.Method == "GET" {
		origin, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			origin = req.Host
		}
		for _, d := range f.Domains {
			if origin == d {
				f.attachHeader(req)
				return
			}
		}
	}
}

func (f *ConfigServerFilter) attachHeader(req *http.Request) {
	req.Header.Set(common.CfgSvrAuthTokenHeader, f.AuthToken)
	log.Debugf("Attached %s header to \"GET %s\"", common.CfgSvrAuthTokenHeader, req.URL.String())
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return
	}
	req.Header.Set(common.CfgSvrClientIPHeader, host)
	log.Debugf("Set %s as %s to \"GET %s\"", common.CfgSvrClientIPHeader, host, req.URL.String())
}
