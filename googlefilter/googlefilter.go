package googlefilter

import (
	"net/http"
	"regexp"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy/filters"
)

var (
	log = golog.LoggerFor("googlefilter")

	DefaultSearchRegex  = `^(www.)?google\..+`
	DefaultCaptchaRegex = `^ipv4.google\..+`
)

// deviceFilterPre does the device-based filtering
type googleFilter struct {
	searchRegex  *regexp.Regexp
	captchaRegex *regexp.Regexp
}

func New(searchRegex string, captchaRegex string) filters.Filter {
	return &googleFilter{
		searchRegex:  regexp.MustCompile(searchRegex),
		captchaRegex: regexp.MustCompile(captchaRegex),
	}
}

func (f *googleFilter) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	f.recordActivity(req)
	return next()
}

func (f *googleFilter) recordActivity(req *http.Request) (sawSearch bool, sawCaptcha bool) {
	if f.searchRegex.MatchString(req.Host) {
		op := ops.Begin("google_search")
		log.Debug("Saw google search")
		op.End()
		return true, false
	}
	if f.captchaRegex.MatchString(req.Host) {
		op := ops.Begin("google_captcha")
		log.Debug("Saw google captcha")
		op.End()
		return false, true
	}
	return false, false
}
