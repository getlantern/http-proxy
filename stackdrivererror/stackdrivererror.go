package stackdrivererror

import (
	"context"
	"fmt"
	"math/rand"

	"google.golang.org/api/option"

	"github.com/getlantern/golog"

	"cloud.google.com/go/errorreporting"
	"github.com/getlantern/http-proxy-lantern/v2/logger"
)

// Reporter is a thin wrapper of Google errorreporting client
type Reporter struct {
	errorClient *errorreporting.Client
	log         golog.Logger
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
	samplePercentage float64, proxyName, externalIP, proxyProtocol string, track string) *Reporter {
	// log := golog.LoggerFor("proxy-stackdriver")
	log := logger.InitializedLogger.SetStdLogger(golog.LoggerFor("proxy-stackdriver"))
	log.Debugf("Enabling stackdriver error reporting for project %v", projectID)
	serviceVersion := track

	// This was a stopgap because at the time of this writing not all proxies know their track.
	if serviceVersion == "" {
		serviceVersion = proxyProtocol
	}
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName:    proxyProtocol,
		ServiceVersion: serviceVersion,
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
