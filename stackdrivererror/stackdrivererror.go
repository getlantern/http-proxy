package stackdrivererror

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"google.golang.org/api/option"

	"github.com/getlantern/golog"

	"cloud.google.com/go/errorreporting"
)

// Close is a function to close the client.
type Close func()

// Enable enables reporting errors to stackdriver.
func Enable(ctx context.Context, projectID, stackdriverCreds string,
	samplePercentage float64, proxyName, externalIP string) Close {
	log := golog.LoggerFor("proxy-stackdriver")
	log.Debugf("Enabling stackdriver error reporting for project %v", projectID)
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName: "lantern-http-proxy-service",
		OnError: func(err error) {
			log.Debugf("Could not log error: %v", err)
		},
	}, option.WithCredentialsFile(stackdriverCreds))
	if err != nil {
		log.Debugf("Error setting up stackdriver error reporting? %v", err)
		return func() {}
	}
	close := func() {
		errorClient.Close()
		errorClient.Flush()
	}

	reporter := func(err error, severity golog.Severity, ctx map[string]interface{}) {
		if severity < golog.ERROR {
			return
		}
		fatal := severity > golog.ERROR || strings.Contains(err.Error(), "panic")
		if !fatal {
			r := rand.Float64()
			if r > samplePercentage {
				log.Debugf("Not in sample. %v less than %v", r, samplePercentage)
				return
			}
		}
		log.Debugf("Reporting error to stackdriver")

		errWithIP := fmt.Errorf("%s: %s on %s(%s)", severity.String(), err.Error(), proxyName, externalIP)
		errorClient.Report(errorreporting.Entry{
			Error: errWithIP,
		})

		if fatal {
			close()
		}
	}

	golog.RegisterReporter(reporter)
	return close
}
