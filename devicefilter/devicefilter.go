package devicefilter

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/context"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/usage"
	"github.com/getlantern/http-proxy/listeners"
)

var log = golog.LoggerFor("devicefilter")

// DeviceFilterPre does the device-based filtering
type DeviceFilterPre struct {
	next http.Handler
}

// DeviceFilterPost cleans up
type DeviceFilterPost struct {
	bl   *blacklist.Blacklist
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
		log.Debugf("No %s header found from %s for request to %v", common.DeviceIdHeader, req.RemoteAddr, req.Host)
	} else {
		// Attached the uid to connection to report stats to redis correctly
		// "conn" in context is previously attached in server.go
		wc := context.Get(req, "conn").(listeners.WrapConn)
		// Sets the ID to the provided key. This message is captured only
		// by the measured wrapper
		wc.ControlMessage("measured", lanternDeviceId)

		limit := uint64(50)
		u := usage.Get(lanternDeviceId)
		uMiB := u / 1024768
		w.Header().Set("XLU", fmt.Sprintf("%d/%d", uMiB, limit))
		if uMiB > limit {
			wc.ControlMessage("bitrate", true)
		}
	}

	f.next.ServeHTTP(w, req)
}

func NewPost(bl *blacklist.Blacklist, next http.Handler) *DeviceFilterPost {
	return &DeviceFilterPost{
		bl:   bl,
		next: next,
	}
}

func (f *DeviceFilterPost) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(common.DeviceIdHeader)
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	f.bl.Succeed(ip)
	f.next.ServeHTTP(w, req)
}
