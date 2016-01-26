package devicefilter

import (
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/context"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/listeners"
)

var log = golog.LoggerFor("devicefilter")

// DeviceFilterPre does the device-based filtering
type DeviceFilterPre struct {
	next http.Handler
}

// DeviceFilterPost cleans up
type DeviceFilterPost struct {
	next http.Handler
}

type optSetter func(f *DeviceFilterPre) error

func NewPre(next http.Handler, setters ...optSetter) (*DeviceFilterPre, error) {
	f := &DeviceFilterPre{
		next: next,
	}
	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *DeviceFilterPre) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	lanternDeviceId := req.Header.Get(common.DeviceIdHeader)

	if lanternDeviceId == "" {
		log.Tracef("No %s header found from %s, usage statistis won't be registered", common.DeviceIdHeader, req.RemoteAddr)
	} else {
		// Attached the uid to connection to report stats to redis correctly
		// "conn" in context is previously attached in server.go
		key := []byte(lanternDeviceId)
		wc := context.Get(req, "conn").(listeners.WrapConn)
		// Sets the ID to the provided key. This message is captured only
		// by the measured wrapper
		wc.ControlMessage("measured", string(key))
	}

	f.next.ServeHTTP(w, req)
}

func NewPost(next http.Handler) *DeviceFilterPost {
	return &DeviceFilterPost{
		next: next,
	}
}

func (f *DeviceFilterPost) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(common.DeviceIdHeader)
	f.next.ServeHTTP(w, req)
}
