package stackdrivererror

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/getlantern/golog"
)

func TestEnable(t *testing.T) {
	log := golog.LoggerFor("stackdrivererror-test")
	ctx := context.Background()

	var percent float64
	percent = 1.0
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	log.Debugf("Using file: %v", credsFile)
	if credsFile != "" {
		close := Enable(ctx, "lantern-http-proxy", credsFile, percent)
		log.Error("Testing error reporting")
		time.Sleep(4 * time.Second)
		close()
	} else {
		log.Debug("Set GOOGLE_APPLICATION_CREDENTIALS to run this test")
	}
}
