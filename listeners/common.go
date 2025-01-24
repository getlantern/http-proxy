package listeners

import (
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/v2/logger"
)

// var log = golog.LoggerFor("listeners")
var log = logger.InitializedLogger.SetStdLogger(golog.LoggerFor("listeners"))
