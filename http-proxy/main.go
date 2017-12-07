package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/vharitonsky/iniflags"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/logging"

	"github.com/getlantern/http-proxy-lantern"
	"github.com/getlantern/http-proxy-lantern/googlefilter"
	"github.com/getlantern/http-proxy-lantern/throttle"
)

var (
	log      = golog.LoggerFor("lantern-proxy")
	revision = "unknown" // overridden by Makefile

	addr                           = flag.String("addr", ":8080", "Address to listen")
	certfile                       = flag.String("cert", "", "Certificate file name")
	cfgSvrAuthToken                = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	cfgSvrDomains                  = flag.String("cfgsvrdomains", "", "Config-server domains on which to attach auth token, separated by comma")
	enableReports                  = flag.Bool("enablereports", false, "Enable stats reporting")
	throttleRefreshInterval        = flag.Duration("throttlerefresh", throttle.DefaultRefreshInterval, "Specifies how frequently to refresh throttling configuration from redis. Defaults to 5 minutes.")
	bordaReportInterval            = flag.Duration("borda-report-interval", 0*time.Second, "How frequently to report errors to borda. Set to 0 to disable reporting.")
	bordaSamplePercentage          = flag.Float64("borda-sample-percentage", 0.0001, "The percentage of devices to report to Borda (0.01 = 1%)")
	bordaBufferSize                = flag.Int("borda-buffer-size", 10000, "Size of borda buffer, caps how many distinct measurements to keep during each submit interval")
	externalIP                     = flag.String("externalip", "", "The external IP of this proxy, used for reporting to Borda")
	help                           = flag.Bool("help", false, "Get usage help")
	https                          = flag.Bool("https", false, "Use TLS for client to proxy communication")
	idleClose                      = flag.Uint64("idleclose", 70, "Time in seconds that an idle connection will be allowed before closing it")
	keyfile                        = flag.String("key", "", "Private key file name")
	logglyToken                    = flag.String("logglytoken", "", "Token used to report to loggly.com, not reporting if empty")
	_                              = flag.Uint64("maxconns", 0, "Max number of simultaneous allowed connections, unused")
	pprofAddr                      = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	pro                            = flag.Bool("pro", false, "Set to true to make this a pro proxy (no bandwidth limiting)")
	proxiedSitesSamplePercentage   = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
	proxiedSitesTrackingId         = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")
	reportingRedisAddr             = flag.String("reportingredis", "", "The address of the reporting Redis instance in \"redis[s]://host:port\" format")
	reportingRedisCA               = flag.String("reportingredisca", "", "Certificate for the CA of Redis instance for reporting")
	reportingRedisClientPK         = flag.String("reportingredisclientpk", "", "Private key for authenticating client to the Redis instance for reporting")
	reportingRedisClientCert       = flag.String("reportingredisclientcert", "", "Certificate for authenticating client to the Redis instance for reporting")
	token                          = flag.String("token", "", "Lantern token")
	tunnelPorts                    = flag.String("tunnelports", "", "Comma seperated list of ports allowed for HTTP CONNECT tunnel. Allow all ports if empty.")
	obfs4Addr                      = flag.String("obfs4-addr", "", "Provide an address here in order to listen with obfs4")
	obfs4Dir                       = flag.String("obfs4-dir", ".", "Directory where obfs4 can store its files")
	kcpConf                        = flag.String("kcpconf", "", "Path to file configuring kcp")
	bench                          = flag.Bool("bench", false, "Set this flag to set up proxy as a benchmarking proxy. This automatically puts the proxy into tls mode and disables auth token authentication.")
	fasttrackDomains               = flag.String("fasttrackdomains", "", "Whitelisted domains, such as the config server, pro server, etc, that should not count towards the bandwidth cap or be throttled, separated by comma")
	tos                            = flag.Int("tos", 0, "Specify a diffserv TOS to prioritize traffic. Defaults to 0 (off)")
	lampshadeAddr                  = flag.String("lampshade-addr", "", "Address at which to listen for lampshade connections. Requires https to be true.")
	version                        = flag.Bool("version", false, "shows the version of the binary")
	versionCheck                   = flag.String("versioncheck", "", "Check if Lantern client is below certain semantic version. No check by default")
	versionCheckRedirectURL        = flag.String("versioncheck-redirect-url", "", "The URL to redirect if client is below certain version. Always used along with versioncheck")
	versionCheckRedirectPercentage = flag.Float64("versioncheck-redirect-percentage", 1, "The percentage of requests to be redirected in version check. Defaults to 1 (100%)")
	googleSearchRegex              = flag.String("google-search-regex", googlefilter.DefaultSearchRegex, "Regex for detecting access to Google Search")
	googleCaptchaRegex             = flag.String("google-captcha-regex", googlefilter.DefaultCaptchaRegex, "Regex for detecting access to Google captcha page")
	domainFront                    = flag.Bool("domainfront", false, "enable support for domain-fronted requests from CloudFront")
)

func main() {
	var err error

	iniflags.Parse()
	if *version {
		fmt.Fprintf(os.Stderr, "%s: commit %s built with %s\n", os.Args[0], revision, runtime.Version())
		return
	}
	if *help {
		flag.Usage()
		return
	}

	if *lampshadeAddr != "" && !*https {
		log.Fatal("Use of lampshade requires https flag to be true")
	}

	// Logging
	// TODO: use real parameters
	err = logging.Init("instanceid", "version", "releasedate", *logglyToken)
	if err != nil {
		log.Fatal(err)
	}

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	if *versionCheck != "" && *versionCheckRedirectURL == "" {
		log.Fatal("version check redirect URL should not empty")
	}

	go periodicallyForceGC()

	p := &proxy.Proxy{
		Addr:                    *addr,
		CertFile:                *certfile,
		CfgSvrAuthToken:         *cfgSvrAuthToken,
		CfgSvrDomains:           *cfgSvrDomains,
		DomainFront:             *domainFront,
		EnableReports:           *enableReports,
		ThrottleRefreshInterval: *throttleRefreshInterval,
		BordaReportInterval:     *bordaReportInterval,
		BordaSamplePercentage:   *bordaSamplePercentage,
		BordaBufferSize:         *bordaBufferSize,
		ExternalIP:              *externalIP,
		HTTPS:                   *https,
		IdleTimeout:             time.Duration(*idleClose) * time.Second,
		KeyFile:                 *keyfile,
		Pro:                     *pro,
		ProxiedSitesSamplePercentage: *proxiedSitesSamplePercentage,
		ProxiedSitesTrackingID:       *proxiedSitesTrackingId,
		ReportingRedisAddr:           *reportingRedisAddr,
		ReportingRedisCA:             *reportingRedisCA,
		ReportingRedisClientPK:       *reportingRedisClientPK,
		ReportingRedisClientCert:     *reportingRedisClientCert,
		Token:                          *token,
		TunnelPorts:                    *tunnelPorts,
		Obfs4Addr:                      *obfs4Addr,
		Obfs4Dir:                       *obfs4Dir,
		KCPConf:                        *kcpConf,
		Benchmark:                      *bench,
		FasttrackDomains:               *fasttrackDomains,
		DiffServTOS:                    *tos,
		LampshadeAddr:                  *lampshadeAddr,
		VersionCheck:                   *versionCheck != "",
		VersionCheckMinVersion:         *versionCheck,
		VersionCheckRedirectURL:        *versionCheckRedirectURL,
		VersionCheckRedirectPercentage: *versionCheckRedirectPercentage,
		GoogleSearchRegex:              *googleSearchRegex,
		GoogleCaptchaRegex:             *googleCaptchaRegex,
	}

	err = p.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func periodicallyForceGC() {
	for {
		time.Sleep(1 * time.Minute)
		debug.FreeOSMemory()
	}
}
