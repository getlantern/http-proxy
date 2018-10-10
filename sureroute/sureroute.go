package sureroute

import (
	"net/http"
	"os"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/filters"
)

const (
	sureroutePath = "/DSA_SLA_test_page.zip"
	surerouteFile = "test_data"
)

var (
	log = golog.LoggerFor("http-proxy-lantern.sureroute")
)

// surerouteMiddleware intercepts sureroute requests and returns a test file
type surerouteMiddleware struct {
}

func New() filters.Filter {
	return &surerouteMiddleware{}
}

func (sr *surerouteMiddleware) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {

	if req.URL.Path != sureroutePath {
		return next(ctx, req)
	}

	log.Debugf("SureRoute test path requested (%s)", req.URL.Path)
	file, err := os.Open(surerouteFile)
	if err != nil {
		log.Debugf("Missing SureRoute file %s: skipping.", surerouteFile)
		return next(ctx, req)
	}

	return filters.ShortCircuit(ctx, req, &http.Response{
		StatusCode: http.StatusOK,
		Body:       file,
	})
}
