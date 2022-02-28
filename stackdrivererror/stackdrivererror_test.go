package stackdrivererror

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/getlantern/zaplog"
)

func TestEnable(t *testing.T) {
	log := zaplog.LoggerFor("stackdrivererror-test")
	ctx := context.Background()

	percent := 1.0
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	log.Infof("Using file: %v", credsFile)
	if credsFile != "" {
		reporter := Enable(ctx, "lantern-http-proxy", credsFile, percent, "fp-testcm-001", "1.1.1.1", "version", "track")
		log.Error("Testing error reporting")
		time.Sleep(2 * time.Second)
		reporter.Close()
	} else {
		log.Info("Set GOOGLE_APPLICATION_CREDENTIALS to run this test")
	}
}
