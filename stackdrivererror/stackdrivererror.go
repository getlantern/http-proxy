package stackdrivererror

import (
	"context"
	"fmt"
	"math/rand"

	"go.uber.org/zap"
	"google.golang.org/api/option"

	"github.com/getlantern/golog"
	"github.com/getlantern/zaplog"

	"cloud.google.com/go/errorreporting"
)

// Reporter is a thin wrapper of Google errorreporting client
type Reporter struct {
	errorClient *errorreporting.Client
	log         *zap.SugaredLogger
	proxyName   string
	externalIP  string
}

func (r *Reporter) Close() {
	r.errorClient.Close()
	r.errorClient.Flush()
}

func (r *Reporter) Report(severity golog.Severity, err error, stack []byte) {
	errWithIP := fmt.Errorf("%s on %s(%s)", err.Error(), r.proxyName, r.externalIP)
	r.log.Debugf("Reporting error to stackdriver: %s", errWithIP)
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
	log := zaplog.LoggerFor("proxy-stackdriver")
	log.Infof("Enabling stackdriver error reporting for project %v", projectID)
	serviceVersion := track

	// This was a stopgap because at the time of this writing not all proxies know their track.
	if serviceVersion == "" {
		serviceVersion = proxyProtocol
	}
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName:    "lantern-http-proxy-service",
		ServiceVersion: serviceVersion,
		OnError: func(err error) {
			log.Infof("Could not log error: %v", err)
		},
	}, option.WithCredentialsFile(stackdriverCreds))
	if err != nil {
		log.Infof("Error setting up stackdriver error reporting? %v", err)
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
				log.Debugf("Not in sample. %v less than %v", r, samplePercentage)
				return
			}
		}

		reporter.Report(severity, err, nil)
	}
	golog.RegisterReporter(gologReporter)

	return reporter
}
