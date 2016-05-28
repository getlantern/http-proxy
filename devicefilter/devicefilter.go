package devicefilter

import (
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/context"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/filter"
	"github.com/getlantern/http-proxy/listeners"

	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("devicefilter")

// deviceFilterPre does the device-based filtering
type deviceFilterPre struct {
}

// deviceFilterPost cleans up
type deviceFilterPost struct {
	bl *blacklist.Blacklist
}

func NewPre() filter.Filter {
	return &deviceFilterPre{}
}

func (f *deviceFilterPre) Apply(w http.ResponseWriter, req *http.Request) (bool, error, string) {
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
		key := []byte(lanternDeviceID)
		wc := context.Get(req, "conn").(listeners.WrapConn)
		// Sets the ID to the provided key. This message is captured only
		// by the measured wrapper
		wc.ControlMessage("measured", string(key))
	}

	return filter.Continue()
}

func NewPost(bl *blacklist.Blacklist) filter.Filter {
	return &deviceFilterPost{
		bl: bl,
	}
}

func (f *deviceFilterPost) Apply(w http.ResponseWriter, req *http.Request) (bool, error, string) {
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(common.DeviceIdHeader)
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	f.bl.Succeed(ip)
	return filter.Continue()
}
