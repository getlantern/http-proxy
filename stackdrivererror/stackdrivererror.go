package stackdrivererror

import (
	"context"
	"fmt"
	"math/rand"

	"google.golang.org/api/option"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/v2/zerologger"

	"cloud.google.com/go/errorreporting"
)

// Reporter is a thin wrapper of Google errorreporting client
type Reporter struct {
	errorClient *errorreporting.Client
	log         zerologger.LoggerWrapper
	proxyName   string
	externalIP  string
}

func (r *Reporter) Close() {
	r.errorClient.Close()
	r.errorClient.Flush()
}

func (r *Reporter) Report(severity golog.Severity, err error, stack []byte) {
	errWithIP := fmt.Errorf("%s on %s(%s)", err.Error(), r.proxyName, r.externalIP)
	r.log.Tracef("Reporting error to stackdriver: %s", errWithIP)
	r.errorClient.Report(errorreporting.Entry{
		Error: errWithIP,
		Stack: stack,
	})
	if severity == golog.FATAL {
		r.Close()
	}
}

// Enable enables golog to report errors to stackdriver and returns the reporter.
func Enable(ctx context.Context, projectID, stackdriverCreds string,
	samplePercentage float64, proxyName, externalIP, proxyProtocol string) *Reporter {
	log := zerologger.Named("proxy-stackdriver")
	log.Debugf("Enabling stackdriver error reporting for project %v", projectID)
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName:    "lantern-http-proxy-service",
		ServiceVersion: proxyProtocol,
		OnError: func(err error) {
			log.Debugf("Could not log error: %v", err)
		},
	}, option.WithCredentialsFile(stackdriverCreds))
	if err != nil {
		log.Debugf("Error setting up stackdriver error reporting? %v", err)
		return nil
	}

	reporter := &Reporter{errorClient, log, proxyName, externalIP}

	gologReporter := func(err error, severity golog.Severity, ctx map[string]interface{}) {
		if severity < golog.ERROR {
			return
		}
		if severity == golog.ERROR {
			r := rand.Float64()
			if r > samplePercentage {
				log.Tracef("Not in sample. %v less than %v", r, samplePercentage)
				return
			}
		}

		reporter.Report(severity, err, nil)
	}
	golog.RegisterReporter(gologReporter)

	return reporter
}
