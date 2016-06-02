package devicefilter

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gorilla/context"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/http-proxy/listeners"

	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/usage"
)

var log = golog.LoggerFor("devicefilter")

// deviceFilterPre does the device-based filtering
type deviceFilterPre struct {
	throttleAfterMiB uint64
}

// deviceFilterPost cleans up
type deviceFilterPost struct {
	bl *blacklist.Blacklist
}

func NewPre(throttleAfterMiB uint64) filters.Filter {
	return &deviceFilterPre{
		throttleAfterMiB: throttleAfterMiB,
	}
}

func (f *deviceFilterPre) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	lanternDeviceID := req.Header.Get(common.DeviceIdHeader)

	if lanternDeviceID == "" {
		log.Debugf("No %s header found from %s for request to %v", common.DeviceIdHeader, req.RemoteAddr, req.Host)
	} else {
		// Attached the uid to connection to report stats to redis correctly
		// "conn" in context is previously attached in server.go
		wc := context.Get(req, "conn").(listeners.WrapConn)
		// Sets the ID to the provided key. This message is captured only
		// by the measured wrapper
		wc.ControlMessage("measured", lanternDeviceID)

		if f.throttleAfterMiB > 0 {
			// Throttling enabled
			u := usage.Get(lanternDeviceID)
			uMiB := u.Bytes / 1024768
			w.Header().Set("XBQ", fmt.Sprintf("%d/%d/%d", uMiB, f.throttleAfterMiB, u.AsOf))
			if uMiB > f.throttleAfterMiB {
				wc.ControlMessage("bitrate", true)
			}
		} else {
			// The below is just testing code that we should remove before actually using this.
			w.Header().Set("XBQ", fmt.Sprintf("%d/%d/%d", rand.Intn(500), 500, time.Now().UnixNano()))
		}
	}

	return next()
}

func NewPost(bl *blacklist.Blacklist) filters.Filter {
	return &deviceFilterPost{
		bl: bl,
	}
}

func (f *deviceFilterPost) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(common.DeviceIdHeader)
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	f.bl.Succeed(ip)
	return next()
}
