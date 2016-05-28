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

type configServerFilter struct {
	*Options
}

func New(opts *Options) filters.Filter {
	if opts.AuthToken == "" || len(opts.Domains) == 0 {
		panic(errors.New("should set both config-server auth token and domains"))
	}
	log.Debugf("Will attach %s header on GET requests to %+v", common.CfgSvrAuthTokenHeader, opts.Domains)
	return &configServerFilter{opts}
}

func (f *configServerFilter) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	// It's unlikely that config-server will add non-GET public endpoint.
	// Bypass all other methods, especially CONNECT (https).
	if req.Method == "GET" {
		for _, d := range f.Domains {
			if req.Host == d {
				req = f.attachHeader(req)
				return next()
			}
		}
	}

	return next()
}

func (f *configServerFilter) attachHeader(req *http.Request) *http.Request {
	req.Header.Set(common.CfgSvrAuthTokenHeader, f.AuthToken)
	log.Debugf("Attached %s header to \"GET %s\"", common.CfgSvrAuthTokenHeader, req.URL.String())
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return req
	}
	req.Header.Set(common.CfgSvrClientIPHeader, host)
	log.Debugf("Set %s as %s to \"GET %s\"", common.CfgSvrClientIPHeader, host, req.URL.String())
	return req
}
