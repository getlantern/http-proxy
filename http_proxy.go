package main

import (
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy/commonfilter"
	"github.com/getlantern/http-proxy/forward"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/logging"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/report"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
)

var (
	testingLocal = false
	log          = golog.LoggerFor("lantern-proxy")

	help                         = flag.Bool("help", false, "Get usage help")
	keyfile                      = flag.String("key", "", "Private key file name")
	certfile                     = flag.String("cert", "", "Certificate file name")
	https                        = flag.Bool("https", false, "Use TLS for client to proxy communication")
	addr                         = flag.String("addr", ":8080", "Address to listen")
	maxConns                     = flag.Uint64("maxconns", 0, "Max number of simultaneous connections allowed connections")
	idleClose                    = flag.Uint64("idleclose", 30, "Time in seconds that an idle connection will be allowed before closing it")
	token                        = flag.String("token", "", "Lantern token")
	enableReports                = flag.Bool("enablereports", false, "Enable stats reporting")
	logglyToken                  = flag.String("logglytoken", "", "Token used to report to loggly.com, not reporting if empty")
	pprofAddr                    = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	proxiedSitesTrackingId       = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")
	proxiedSitesSamplePercentage = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
)

func main() {
	var err error

	_ = flag.CommandLine.Parse(os.Args[1:])
	if *help {
		flag.Usage()
		return
	}

	// Logging
	// TODO: use real parameters
	err = logging.Init("instanceid", "version", "releasedate", *logglyToken)
	if err != nil {
		log.Error(err)
	}

	// Reporting
	if *enableReports {
		redisAddr := os.Getenv("REDIS_PRODUCTION_URL")
		if redisAddr == "" {
			redisAddr = "127.0.0.1:6379"
		}
		rp, err := report.NewRedisReporter(redisAddr)
		if err != nil {
			log.Errorf("Error connecting to redis: %v", err)
		} else {
			measured.Start(20*time.Second, rp)
			defer measured.Stop()
		}
	}

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	// Middleware
	forwarder, err := forward.New(nil, forward.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second))
	if err != nil {
		log.Error(err)
	}

	httpConnect, err := httpconnect.New(forwarder, httpconnect.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second))
	if err != nil {
		log.Error(err)
	}

	commonFilter, err := commonfilter.New(httpConnect,
		testingLocal,
		commonfilter.SetException("127.0.0.1:7300"),
	)
	if err != nil {
		log.Error(err)
	}

	deviceFilterPost := devicefilter.NewPost(commonFilter)

	analyticsFilter := analytics.New(*proxiedSitesTrackingId, *proxiedSitesSamplePercentage, deviceFilterPost)

	deviceFilterPre, err := devicefilter.NewPre(analyticsFilter)
	if err != nil {
		log.Error(err)
	}

	tokenFilter, err := tokenfilter.New(deviceFilterPre, tokenfilter.TokenSetter(*token))
	if err != nil {
		log.Error(err)
	}

	srv := server.NewServer(tokenFilter)

	// Add net.Listener wrappers for inbound connections
	if *enableReports {
		srv.AddListenerWrappers(
			// Measure connections
			func(ls net.Listener) net.Listener {
				return listeners.NewMeasuredListener(ls, 100*time.Millisecond)
			},
		)
	}
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, time.Duration(*idleClose)*time.Second)
		},
		// Preprocess connection to issue custom errors before they are passed to the server
		func(ls net.Listener) net.Listener {
			return lanternlisteners.NewPreprocessorListener(ls)
		},
	)

	initMimic := func(addr string) {
		mimic.SetServerAddr(addr)
	}

	if *https {
		err = srv.ServeHTTPS(*addr, *keyfile, *certfile, initMimic)
	} else {
		err = srv.ServeHTTP(*addr, initMimic)
	}
	if err != nil {
		log.Errorf("Error serving: %v", err)
	}
}
