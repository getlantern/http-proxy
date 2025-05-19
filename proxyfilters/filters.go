package proxyfilters

import (
	"fmt"
	"net/http"

	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/v3/filters"
)

var log = golog.LoggerFor("http-proxy.filters")

func fail(cs *filters.ConnectionState, req *http.Request, statusCode int, description string, params ...interface{}) (*http.Response, *filters.ConnectionState, error) {
	msg := fmt.Sprintf("Filter fail: %s", description)
	log.Errorf(msg, params...)
	return filters.Fail(cs, req, statusCode, errors.New(msg, params...))
}
