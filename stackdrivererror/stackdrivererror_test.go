package stackdrivererror

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/getlantern/http-proxy-lantern/v2/zerologger"
)

func TestEnable(t *testing.T) {
	log := zerologger.Named("stackdrivererror-test")
	ctx := context.Background()

	percent := 1.0
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	log.Debugf("Using file: %v", credsFile)
	if credsFile != "" {
		reporter := Enable(ctx, "lantern-http-proxy", credsFile, percent, "fp-testcm-001", "1.1.1.1", "version")
		log.Error("Testing error reporting")
		time.Sleep(2 * time.Second)
		reporter.Close()
	} else {
		log.Debug("Set GOOGLE_APPLICATION_CREDENTIALS to run this test")
	}
}
