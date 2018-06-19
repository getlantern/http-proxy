package stackdrivererror

import (
	"context"
	"log"
	"math/rand"

	"google.golang.org/api/option"

	"github.com/getlantern/golog"

	"cloud.google.com/go/errorreporting"
)

// Close is a function to close the client.
type Close func()

// Enable enables reporting errors to stackdriver.
func Enable(ctx context.Context, projectID, stackdriverCreds string, samplePercentage float64) Close {
	log.Printf("Enabling stackdriver error reporting for project %v", projectID)
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName: "lantern-http-proxy-service",
		OnError: func(err error) {
			log.Printf("Could not log error: %v", err)
		},
	}, option.WithCredentialsFile(stackdriverCreds))
	if err != nil {
		log.Printf("Error setting up stackdriver error reporting? %v", err)
		return func() {}
	}

	var reporter = func(err error, linePrefix string, severity golog.Severity, ctx map[string]interface{}) {
		if severity == golog.ERROR || severity == golog.FATAL {
			r := rand.Float64()
			if r > samplePercentage {
				log.Printf("Not in sample. %v less than %v", r, samplePercentage)
				return
			}
			log.Println("Reporting error to stackdriver")
			errorClient.Report(errorreporting.Entry{
				Error: err,
			})
		}
	}

	golog.RegisterReporter(reporter)
	return func() {
		errorClient.Close()
		errorClient.Flush()
	}
}
