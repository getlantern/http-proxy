// Add required headers to config-server requests.
// Ref https://github.com/getlantern/config-server/issues/4

package cfgsvrfilter

import (
	"net/http"

	"github.com/getlantern/golog"
)

const (
	cfgSvrAuthTokenHeader = "X-Lantern-Config-Auth-Token"
	cfgSvrClientIPHeader  = "X-Lantern-Config-Client-IP"
)

var log = golog.LoggerFor("cfgsvrfilter")

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
		panic("should set both config-server auth token and domains")
	}

	log.Debugf("Will attach %s header on GET requests to %+v", cfgSvrAuthTokenHeader, f.domains)
	return f, nil
}

func (f *ConfigServerFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
	req.Header.Set(cfgSvrAuthTokenHeader, f.authToken)
	log.Debugf("Attached %s header to \"GET %s\"", cfgSvrAuthTokenHeader, req.URL.String())
	req.Header.Set(cfgSvrClientIPHeader, req.RemoteAddr)
	log.Debugf("Set %s as %s to \"GET %s\"", cfgSvrClientIPHeader, req.RemoteAddr, req.URL.String())
	return req
}
