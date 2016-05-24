package main

import (
	"encoding/base64"
	"encoding/binary"
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/vharitonsky/iniflags"

	borda "github.com/getlantern/borda/client"
	"github.com/getlantern/context"
	"github.com/getlantern/golog"
	"github.com/getlantern/measured"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy/commonfilter"
	"github.com/getlantern/http-proxy/forward"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/logging"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/opsfilter"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/profilter"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
)

var (
	testingLocal = false
	log          = golog.LoggerFor("lantern-proxy")

	addr                         = flag.String("addr", ":8080", "Address to listen")
	certfile                     = flag.String("cert", "", "Certificate file name")
	cfgSvrAuthToken              = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	cfgSvrDomains                = flag.String("cfgsvrdomains", "", "Config-server domains on which to attach auth token, separated by comma")
	enablePro                    = flag.Bool("enablepro", false, "Enable Lantern Pro support")
	enableReports                = flag.Bool("enablereports", false, "Enable stats reporting")
	bordaReportInterval          = flag.Duration("borda-report-interval", 30*time.Second, "How frequently to report errors to borda. Set to 0 to disable reporting.")
	bordaSamplePercentage        = flag.Float64("borda-sample-percentage", 0.001, "The percentage of devices to report to Borda (0.01 = 1%)")
	help                         = flag.Bool("help", false, "Get usage help")
	https                        = flag.Bool("https", false, "Use TLS for client to proxy communication")
	idleClose                    = flag.Uint64("idleclose", 30, "Time in seconds that an idle connection will be allowed before closing it")
	keyfile                      = flag.String("key", "", "Private key file name")
	logglyToken                  = flag.String("logglytoken", "", "Token used to report to loggly.com, not reporting if empty")
	maxConns                     = flag.Uint64("maxconns", 0, "Max number of simultaneous connections allowed connections")
	pprofAddr                    = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	proxiedSitesSamplePercentage = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
	proxiedSitesTrackingId       = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")
	redisAddr                    = flag.String("redis", "127.0.0.1:6379", "Redis address in \"host:port\" format")
	redisCA                      = flag.String("redisca", "", "Certificate for redislabs's CA")
	redisClientPK                = flag.String("redisclientpk", "garantia_user_private.key", "Private key for authenticating client to redis's stunnel")
	redisClientCert              = flag.String("redisclientcert", "garantia_user.crt", "Certificate for authenticating client to redis's stunnel")
	serverId                     = flag.String("serverid", "", "Server Id required for Pro-supporting servers")
	token                        = flag.String("token", "", "Lantern token")
	tunnelPorts                  = flag.String("tunnelports", "", "Comma seperated list of ports allowed for HTTP CONNECT tunnel. Allow all ports if empty.")
	obfs4Addr                    = flag.String("obfs4-addr", "", "Provide an address here in order to listen with obfs4")
	obfs4Dir                     = flag.String("obfs4-dir", ".", "Directory where obfs4 can store its files")
)

func main() {
	var err error

	iniflags.Parse()
	if *help {
		flag.Usage()
		return
	}

	// Logging
	// TODO: use real parameters
	err = logging.Init("instanceid", "version", "releasedate", *logglyToken)
	if err != nil {
		log.Fatal(err)
	}

	context.PutGlobal("app", "http-proxy")

	redisOpts := &redis.Options{
		RedisURL:       *redisAddr,
		RedisCAFile:    *redisCA,
		ClientPKFile:   *redisClientPK,
		ClientCertFile: *redisClientCert,
	}
	// Reporting
	if *enableReports {
		rp, err := redis.NewMeasuredReporter(redisOpts)
		if err != nil {
			log.Fatalf("Error creating mesured reporter: %v", err)
		}
		measured.Start(time.Minute, rp)
		defer measured.Stop()
	}

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	// Configure borda
	if *bordaReportInterval > 0 {
		bordaClient := borda.NewClient(&borda.Options{
			BatchInterval: *bordaReportInterval,
		})
		reportToBorda := bordaClient.ReducingSubmitter("proxy_results", 10000, func(existingValues map[string]float64, newValues map[string]float64) {
			for key, value := range newValues {
				existingValues[key] += value
			}
		})
		ops.RegisterReporter(func(failure error, ctx map[string]interface{}) {
			// Sample a subset of device ids
			deviceID := ctx["device_id"]
			if deviceID == nil {
				log.Debugf("No device id, not reporting measurement")
				return
			}
			deviceIDBytes, base64Err := base64.StdEncoding.DecodeString(deviceID.(string))
			if base64Err != nil {
				log.Debugf("Error decoding base64 deviceID: %v", base64Err)
				return
			}
			var deviceIDInt uint64
			if len(deviceIDBytes) < 4 {
				log.Debugf("DeviceID too small: %v", base64Err)
			} else if len(deviceIDBytes) < 8 {
				deviceIDInt = uint64(binary.BigEndian.Uint32(deviceIDBytes))
			} else {
				deviceIDInt = binary.BigEndian.Uint64(deviceIDBytes)
			}
			if deviceIDInt%uint64(1 / *bordaSamplePercentage) != 0 {
				log.Trace("DeviceID not being sampled")
			}

			values := map[string]float64{}
			if failure != nil {
				values["error_count"] = 1
			} else {
				values["success_count"] = 1
			}
			reportErr := reportToBorda(values, ctx)
			if reportErr != nil {
				log.Errorf("Error reporting error to borda: %v", reportErr)
			}
		})
	}

	// Set up a blacklist
	bl := blacklist.New(30*time.Second, 10, 6*time.Hour)

	// Middleware
	forwarder, err := forward.New(nil, forward.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	var nextFilter http.Handler = forwarder

	if *tunnelPorts != "" {
		nextFilter, err = httpconnect.New(forwarder,
			httpconnect.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second),
			httpconnect.AllowedPortsFromCSV(*tunnelPorts))
	} else {
		nextFilter, err = httpconnect.New(forwarder,
			httpconnect.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second))
	}
	if err != nil {
		log.Fatal(err)
	}

	if *cfgSvrAuthToken != "" || *cfgSvrDomains != "" {
		domains := strings.Split(*cfgSvrDomains, ",")
		nextFilter, err = configserverfilter.New(nextFilter,
			configserverfilter.AuthToken(*cfgSvrAuthToken),
			configserverfilter.Domains(domains))
		if err != nil {
			log.Fatal(err)
		}
	}

	pingFilter := ping.New(nextFilter)

	commonFilter, err := commonfilter.New(pingFilter,
		testingLocal,
		commonfilter.SetException("127.0.0.1:7300"),
	)
	if err != nil {
		log.Fatal(err)
	}

	deviceFilterPost := devicefilter.NewPost(bl, commonFilter)

	analyticsFilter := analytics.New(*proxiedSitesTrackingId, *proxiedSitesSamplePercentage, deviceFilterPost)

	deviceFilterPre, err := devicefilter.NewPre(analyticsFilter)
	if err != nil {
		log.Fatal(err)
	}

	tokenFilter, err := tokenfilter.New(deviceFilterPre, tokenfilter.TokenSetter(*token))
	if err != nil {
		log.Fatal(err)
	}

	var top http.Handler = tokenFilter

	// Pro support
	if *enablePro {
		if *serverId == "" {
			log.Fatal("Enabling Pro requires setting the \"serverid\" flag")
		}
		log.Debug("This proxy is configured to support Lantern Pro")
		proFilter, err2 := profilter.New(tokenFilter,
			profilter.RedisConfigSetter(redisOpts, *serverId),
		)
		if err2 != nil {
			log.Fatal(err2)
		}
		top = proFilter
	}

	srv := server.NewServer(opsfilter.New(top))

	// Only allow connections from remote IPs that are not blacklisted
	srv.Allow = bl.OnConnect

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

	onAddress := func(addr string) {
		proxyHost, proxyPort, err2 := net.SplitHostPort(addr)
		if err2 == nil {
			context.PutGlobal("proxy_host", proxyHost)
			context.PutGlobal("proxy_port", proxyPort)
		}
		mimic.SetServerAddr(addr)
	}

	if *obfs4Addr != "" {
		l, err := obfs4listener.NewListener(*obfs4Addr, *obfs4Dir)
		if err != nil {
			log.Fatalf("Unable to listen with obfs4: %v", err)
		}
		go func() {
			err := srv.Serve(l, func(addr string) {
				log.Debugf("obfs4 listening at %v", addr)
			})
			if err != nil {
				log.Fatalf("Error serving OBFS4: %v", err)
			}
		}()
	}
	if *https {
		err = srv.ListenAndServeHTTPS(*addr, *keyfile, *certfile, onAddress)
	} else {
		err = srv.ListenAndServeHTTP(*addr, onAddress)
	}
	if err != nil {
		log.Errorf("Error serving HTTP(S): %v", err)
	}
}
