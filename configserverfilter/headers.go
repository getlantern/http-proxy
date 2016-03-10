// Add required headers to config-server requests.
// Ref https://github.com/getlantern/config-server/issues/4

package configserverfilter

import (
	"errors"
	"net"
	"net/http"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("configserverfilter")

type ConfigServerFilter struct {
	next      http.Handler
	authToken string
	domains   []string
}

type optSetter func(f *ConfigServerFilter)

func AuthToken(token string) optSetter {
	return func(f *ConfigServerFilter) {
		f.authToken = token
	}
}

func Domains(domains []string) optSetter {
	return func(f *ConfigServerFilter) {
		f.domains = domains
	}
}

func New(next http.Handler, setters ...optSetter) (*ConfigServerFilter, error) {
	f := &ConfigServerFilter{
		next: next,
	}

	for _, s := range setters {
		s(f)
	}

	if f.authToken == "" || len(f.domains) == 0 {
		return nil, errors.New("should set both config-server auth token and domains")
	}

	log.Debugf("Will attach %s header on GET requests to %+v", common.CfgSvrAuthTokenHeader, f.domains)
	return f, nil
}

func (f *ConfigServerFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// It's unlikely that config-server will add non-GET public endpoint.
	// Bypass all other methods, especially CONNECT (https).
	if req.Method == "GET" {
		for _, d := range f.domains {
			if req.Host == d {
				req = f.attachHeader(req)
				goto next
			}
		}
	}
next:
	f.next.ServeHTTP(w, req)
}

func (f *ConfigServerFilter) attachHeader(req *http.Request) *http.Request {
	req.Header.Set(common.CfgSvrAuthTokenHeader, f.authToken)
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
