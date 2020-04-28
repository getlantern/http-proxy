package stackdrivererror

import (
	"context"
	"errors"
	"math/rand"

	"google.golang.org/api/option"

	"github.com/getlantern/golog"

	"cloud.google.com/go/errorreporting"
)

// Close is a function to close the client.
type Close func()

// Enable enables reporting errors to stackdriver.
func Enable(ctx context.Context, projectID, stackdriverCreds string,
	samplePercentage float64, externalIP string) Close {
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

	var reporter = func(err error, severity golog.Severity, ctx map[string]interface{}) {
		if severity == golog.ERROR || severity == golog.FATAL {
			if severity == golog.ERROR {
				r := rand.Float64()
				if r > samplePercentage {
					log.Debugf("Not in sample. %v less than %v", r, samplePercentage)
					return
				}
			}
			log.Debugf("Reporting error to stackdriver")

			errWithIP := errors.New(err.Error() + " to: " + externalIP)
			errorClient.Report(errorreporting.Entry{
				Error: errWithIP,
			})

			if severity == golog.FATAL {
				errorClient.Close()
			}
		}
	}

	golog.RegisterReporter(reporter)
	return func() {
		errorClient.Close()
		errorClient.Flush()
	}
}
