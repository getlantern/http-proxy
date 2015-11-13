package devicefilter

import (
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/context"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/listeners"
)

const (
	deviceIdHeader = "X-Lantern-Device-Id"
)

var log = golog.LoggerFor("devicefilter")

type DeviceFilter struct {
	next http.Handler
}

type optSetter func(f *DeviceFilter) error

func New(next http.Handler, setters ...optSetter) (*DeviceFilter, error) {
	f := &DeviceFilter{
		next: next,
	}
	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *DeviceFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	lanternDeviceId := req.Header.Get(deviceIdHeader)

	if lanternDeviceId == "" {
		log.Tracef("No %s header found from %s, usage statistis won't be registered", deviceIdHeader, req.RemoteAddr)
	} else {
		// Attached the uid to connection to report stats to redis correctly
		// "conn" in context is previously attached in server.go
		key := []byte(lanternDeviceId)
		wc := context.Get(req, "conn").(listeners.WrapConn)
		// Sets the ID to the provided key. This message is captured only
		// by the measured wrapper
		wc.ControlMessage("measured", string(key))

		req.Header.Del(deviceIdHeader)
	}

	f.next.ServeHTTP(w, req)
}
