package googlefilter

import (
	"net/http"
	"testing"

	"github.com/getlantern/proxy/filters"
	"github.com/stretchr/testify/assert"
)

func TestRecordGoogleActivity(t *testing.T) {
	f := New(DefaultSearchRegex, DefaultCaptchaRegex).(*googleFilter)

	check := func(host string, expectSearch bool, expectCaptcha bool) {
		req, _ := http.NewRequest(http.MethodGet, "https://"+host, nil)
		sawSearch, sawCaptcha := f.recordActivity(req)
		assert.Equal(t, expectSearch, sawSearch, "search on %v", host)
		assert.Equal(t, expectCaptcha, sawCaptcha, "captcha on %v", host)
	}

	check("osnews.com", false, false)
	check("mail.google.com", false, false)
	check("google.com", true, false)
	check("www.google.com", true, false)
	check("google.co.jp", true, false)
	check("www.google.co.jp", true, false)
	check("ipv4.google.com", false, true)
	check("ipv4.google.co.jp", false, true)
}

func TestApply(t *testing.T) {
	f := New(DefaultSearchRegex, DefaultCaptchaRegex).(*googleFilter)
	req, _ := http.NewRequest(http.MethodGet, "https://google.com", nil)
	_, _, err := f.Apply(filters.BackgroundContext(), req, func(ctx filters.Context, req *http.Request) (*http.Response, filters.Context, error) {
		return nil, ctx, nil
	})

	assert.NoError(t, err)
}
