package stackdrivererror

import (
	"context"
	"log"

	"google.golang.org/api/option"

	"github.com/getlantern/golog"

	"cloud.google.com/go/errorreporting"
)

var errorClient *errorreporting.Client

// Enable enables reporting errors to stackdriver.
func Enable(ctx context.Context, projectID, stackdriverCreds string) {
	log.Printf("Enabling stackdriver error reporting for project %v", projectID)
	var err error
	errorClient, err = errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName: "lantern-http-proxy-service",
		OnError: func(err error) {
			log.Printf("Could not log error: %v", err)
		},
	}, option.WithCredentialsFile(stackdriverCreds))
	if err != nil {
		log.Printf("Error setting up stackdriver error reporting? %v", err)
		return
	}

	var reporter = func(err error, linePrefix string, severity golog.Severity, ctx map[string]interface{}) {
		if severity == golog.ERROR || severity == golog.FATAL {
			log.Println("Reporting error to stackdriver")
			errorClient.Report(errorreporting.Entry{
				Error: err,
			})
		}
	}

	golog.RegisterReporter(reporter)
}

// Close closes the stackdriver error reporting client.
func Close() {
	if errorClient != nil {
		errorClient.Close()
		errorClient.Flush()
	}
}
