package proxyfilters

import (
	"net/http"

	"github.com/getlantern/errors"
	"github.com/getlantern/http-proxy-lantern/v2/logger"
	"github.com/getlantern/proxy/v3/filters"
)

var log = logger.InitLogger("http-proxy.filters")

func fail(cs *filters.ConnectionState, req *http.Request, statusCode int, description string, params ...interface{}) (*http.Response, *filters.ConnectionState, error) {
	log.Errorf("Filter fail: "+description, params...)
	return filters.Fail(cs, req, statusCode, errors.New(description, params...))
}
