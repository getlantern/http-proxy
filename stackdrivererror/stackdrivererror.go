package stackdrivererror

import (
	"context"
	"fmt"
	"math/rand"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"

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

func (r *Reporter) Report(severity zapcore.Level, err error, stack []byte) {
	errWithIP := fmt.Errorf("%s on %s(%s)", err.Error(), r.proxyName, r.externalIP)
	r.log.Debugf("Reporting error to stackdriver: %s", errWithIP)
	r.errorClient.Report(errorreporting.Entry{
		Error: errWithIP,
		Stack: stack,
	})
	if severity >= zapcore.FatalLevel {
		r.Close()
	}
}

// Enable enables golog to report errors to stackdriver and returns the reporter.
func Enable(projectID, stackdriverCreds string,
	samplePercentage float64, proxyName, externalIP, proxyProtocol string, track string) (*Reporter, error) {
	log := zaplog.LoggerFor("proxy-stackdriver")
	log.Infof("Enabling stackdriver error reporting for project %v", projectID)
	serviceVersion := track

	// This was a stopgap because at the time of this writing not all proxies know their track.
	if serviceVersion == "" {
		serviceVersion = proxyProtocol
	}
	errorClient, err := errorreporting.NewClient(context.Background(), projectID, errorreporting.Config{
		ServiceName:    "lantern-http-proxy-service",
		ServiceVersion: serviceVersion,
		OnError: func(err error) {
			log.Infof("Could not log error: %v", err)
		},
	}, option.WithCredentialsFile(stackdriverCreds))
	if err != nil {
		return nil, fmt.Errorf("error setting up stackdriver error reporting? %w", err)
	}

	reporter := &Reporter{errorClient, log, proxyName, externalIP}

	zapReporter := func(entry zapcore.Entry) error { //func(err error, severity zapcore.Level, ctx map[string]interface{}) {
		if entry.Level < zapcore.WarnLevel {
			return nil
		}
		r := rand.Float64()
		if r > samplePercentage {
			log.Debugf("Not in sample. %v less than %v", r, samplePercentage)
			return nil
		}

		reporter.Report(entry.Level, err, nil)
		return nil
	}
	zaplog.RegisterHook(zapReporter)
	return reporter, nil
}
