// Tunnel ports filter compares req.Host of HTTP CONNECT requests with predefined
// set of ports. It allows only matched ones to pass, and responds "403 Port not
// allowed" otherwise.

package tunnelportsfilter

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/getlantern/golog"
)

var log = golog.LoggerFor("tunnelportsfilter")

type tunnelPortsFilter struct {
	next         http.Handler
	allowedPorts []int
}

type optSetter func(f *tunnelPortsFilter) error

func AllowedPorts(ports []int) optSetter {
	return func(f *tunnelPortsFilter) error {
		f.allowedPorts = ports
		return nil
	}
}

func New(next http.Handler, setters ...optSetter) (http.Handler, error) {
	if next == nil {
		return nil, errors.New("Next handler is not defined (nil)")
	}
	f := &tunnelPortsFilter{next: next}

	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *tunnelPortsFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		f.next.ServeHTTP(w, req)
	}

	log.Tracef("Checking CONNECT tunnel to %s against allowed ports %v", req.Host, f.allowedPorts)
	idx := strings.LastIndex(req.Host, ":")
	if idx < 0 || idx >= len(req.Host)-1 {
		// CONNECT request should always include port in req.Host.
		// Ref https://tools.ietf.org/html/rfc2817#section-5.2.
		f.ServeError(w, req, http.StatusBadRequest, "No port field in Request-URI / Host header")
		return
	}
	port, err := strconv.Atoi(req.Host[idx+1:])
	if err != nil {
		f.ServeError(w, req, http.StatusBadRequest, "Invalid port")
		return
	}

	for _, p := range f.allowedPorts {
		if port == p {
			f.next.ServeHTTP(w, req)
			return
		}
	}
	f.ServeError(w, req, http.StatusForbidden, "Port not allowed")
}

func (f *tunnelPortsFilter) ServeError(w http.ResponseWriter, req *http.Request, statusCode int, reason string) {
	log.Debugf("CONNECT to %s: %d %s", req.Host, statusCode, reason)
	w.WriteHeader(statusCode)
	w.Write([]byte(reason))
}
