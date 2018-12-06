package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/vharitonsky/iniflags"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy-lantern"
	"github.com/getlantern/http-proxy-lantern/googlefilter"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/stackdrivererror"
	"github.com/getlantern/http-proxy-lantern/throttle"
)

var (
	log      = golog.LoggerFor("lantern-proxy")
	revision = "unknown" // overridden by Makefile

	hostname, _ = os.Hostname()

	fasttrack = "adyenpayments.com,adyen.com,stripe.com,paymentwall.com,alipay.com,app-measurement.com,fastworldpay.com,firebaseremoteconfig.googleapis.com,firebaseio.com,getlantern.org,lantern.io,innovatelabs.io,getiantem.org,lantern-pro-server.herokuapp.com,lantern-pro-server-staging.herokuapp.com,optimizely.com"

	addr                               = flag.String("addr", ":8080", "Address to listen with HTTP(S)")
	multiplexAddr                      = flag.String("multiplexaddr", "", "Multiplexed address at which to listen with HTTP(S)")
	certfile                           = flag.String("cert", "", "Certificate file name")
	cfgSvrAuthToken                    = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	cfgSvrDomains                      = flag.String("cfgsvrdomains", "", "Config-server domains on which to attach auth token, separated by comma")
	connectOKWaitsForUpstream          = flag.Bool("connect-ok-waits-for-upstream", false, "Set to true to wait for upstream connection before responding OK to CONNECT requests")
	enableReports                      = flag.Bool("enablereports", false, "Enable stats reporting")
	throttleRefreshInterval            = flag.Duration("throttlerefresh", throttle.DefaultRefreshInterval, "Specifies how frequently to refresh throttling configuration from redis. Defaults to 5 minutes.")
	throttleThreshold                  = flag.Int64("throttlethreshold", 0, "Set to a positive value to force a specific throttle threshold in bytes (rather than using one from Redis)")
	throttleRate                       = flag.Int64("throttlerate", 0, "Set to a positive value to force a specific throttle rate in bytes/second (rather than using one from Redis)")
	bordaReportInterval                = flag.Duration("borda-report-interval", 0*time.Second, "How frequently to report errors to borda. Set to 0 to disable reporting.")
	bordaSamplePercentage              = flag.Float64("borda-sample-percentage", 0.0001, "The percentage of devices to report to Borda (0.01 = 1%)")
	bordaBufferSize                    = flag.Int("borda-buffer-size", 10000, "Size of borda buffer, caps how many distinct measurements to keep during each submit interval")
	externalIP                         = flag.String("externalip", "", "The external IP of this proxy, used for reporting to Borda")
	help                               = flag.Bool("help", false, "Get usage help")
	https                              = flag.Bool("https", false, "Use TLS for client to proxy communication")
	idleClose                          = flag.Uint64("idleclose", 70, "Time in seconds that an idle connection will be allowed before closing it")
	keyfile                            = flag.String("key", "", "Private key file name")
	logglyToken                        = flag.String("logglytoken", "", "Token used to report to loggly.com, not reporting if empty")
	_                                  = flag.Uint64("maxconns", 0, "Max number of simultaneous allowed connections, unused")
	pprofAddr                          = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	pro                                = flag.Bool("pro", false, "Set to true to make this a pro proxy (no bandwidth limiting unless forced throttling)")
	proxiedSitesSamplePercentage       = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
	proxiedSitesTrackingId             = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")
	reportingRedisAddr                 = flag.String("reportingredis", "", "The address of the reporting Redis instance in \"redis[s]://host:port\" format")
	reportingRedisCA                   = flag.String("reportingredisca", "", "Certificate for the CA of Redis instance for reporting")
	reportingRedisClientPK             = flag.String("reportingredisclientpk", "", "Private key for authenticating client to the Redis instance for reporting")
	reportingRedisClientCert           = flag.String("reportingredisclientcert", "", "Certificate for authenticating client to the Redis instance for reporting")
	token                              = flag.String("token", "", "Lantern token")
	tunnelPorts                        = flag.String("tunnelports", "", "Comma seperated list of ports allowed for HTTP CONNECT tunnel. Allow all ports if empty.")
	obfs4Addr                          = flag.String("obfs4-addr", "", "Provide an address here in order to listen with obfs4")
	obfs4MultiplexAddr                 = flag.String("obfs4-multiplexaddr", "", "Provide an address here in order to listen with multiplexed obfs4")
	obfs4Dir                           = flag.String("obfs4-dir", ".", "Directory where obfs4 can store its files")
	obfs4HandshakeConcurrency          = flag.Int("obfs4-handshake-concurrency", obfs4listener.DefaultHandshakeConcurrency, "How many concurrent OBFS4 handshakes to process")
	obfs4MaxPendingHandshakesPerClient = flag.Int("obfs4-max-pending-handshakes-per-client", obfs4listener.DefaultMaxPendingHandshakesPerClient, "How many pending OBFS4 handshakes to allow per client")
	obfs4HandshakeTimeout              = flag.Duration("obfs4-handshake-timeout", obfs4listener.DefaultHandshakeTimeout, "How long to wait before timing out an OBFS4 handshake")
	kcpConf                            = flag.String("kcpconf", "", "Path to file configuring kcp")
	enhttpAddr                         = flag.String("enhttp-addr", "", "Address at which to accept encapsulated HTTP requests")
	enhttpServerURL                    = flag.String("enhttp-server-url", "", "specify a full URL for domain-fronting to this server with enhttp, required for sticky routing with CloudFront")
	enhttpReapIdleTime                 = flag.Duration("enhttp-reapidletime", time.Duration(*idleClose)*time.Second, "configure how long enhttp connections are allowed to remain idle before being forcibly closed")
	bench                              = flag.Bool("bench", false, "Set this flag to set up proxy as a benchmarking proxy. This automatically puts the proxy into tls mode and disables auth token authentication.")
	fasttrackDomains                   = flag.String("fasttrackdomains", fasttrack, "Whitelisted domains, such as the config server, pro server, etc, that should not count towards the bandwidth cap or be throttled, separated by comma")
	tos                                = flag.Int("tos", 0, "Specify a diffserv TOS to prioritize traffic. Defaults to 0 (off)")
	lampshadeAddr                      = flag.String("lampshade-addr", "", "Address at which to listen for lampshade connections. Requires https to be true.")
	version                            = flag.Bool("version", false, "shows the version of the binary")
	versionCheck                       = flag.String("versioncheck", "", "Check if Lantern client matches the semantic version range, like \"< 3.1.1\" or \"<= 3.x\". No check by default")
	versionCheckRedirectURL            = flag.String("versioncheck-redirect-url", "", "The URL to redirect if client is below certain version. Always used along with versioncheck")
	versionCheckRedirectPercentage     = flag.Float64("versioncheck-redirect-percentage", 1, "The percentage of requests to be redirected in version check. Defaults to 1 (100%)")
	googleSearchRegex                  = flag.String("google-search-regex", googlefilter.DefaultSearchRegex, "Regex for detecting access to Google Search")
	googleCaptchaRegex                 = flag.String("google-captcha-regex", googlefilter.DefaultCaptchaRegex, "Regex for detecting access to Google captcha page")
	blacklistMaxIdleTime               = flag.Duration("blacklist-max-idle-time", 2*time.Minute, "How long to wait for an HTTP request before considering a connection failed for blacklisting")
	blacklistMaxConnectInterval        = flag.Duration("blacklist-max-connect-interval", 10*time.Second, "Successive connection attempts within this interval will be treated as a single attempt for blacklisting")
	blacklistAllowedFailures           = flag.Int("blacklist-allowed-failures", 100, "The number of failed connection attempts we tolerate before blacklisting an IP address")
	blacklistExpiration                = flag.Duration("blacklist-expiration", 6*time.Hour, "How long to wait before removing an ip from the blacklist")
	proxyName                          = flag.String("proxyname", hostname, "The name of this proxy (defaults to hostname)")
	bbrUpstreamProbeURL                = flag.String("bbrprobeurl", "", "optional URL to probe for upstream BBR bandwidth estimates")
	stackdriverProjectID               = flag.String("stackdriver-project-id", "lantern-http-proxy", "Optional project ID for stackdriver error reporting as in http-proxy-lantern")
	stackdriverCreds                   = flag.String("stackdriver-creds", "/home/lantern/lantern-stackdriver.json", "Optional full json file path containing stackdriver credentials")
	stackdriverSamplePercentage        = flag.Float64("stackdriver-sample-percentage", 0.003, "The percentage of devices to report to Stackdriver (0.01 = 1%)")
	quicAddr                           = flag.String("quic-addr", "", "Address at which to listen for QUIC connections.")
	pcapDir                            = flag.String("pcap-dir", "/tmp", "Directory in which to save pcaps")
	pcapIPs                            = flag.Int("pcap-ips", 0, "The number of IP addresses for which to capture packets")
	pcapsPerIP                         = flag.Int("pcaps-per-ip", 0, "The number of packets to capture for each IP address")
)

func main() {
	ctx := context.Background()

	iniflags.SetAllowUnknownFlags(true)
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

	if *stackdriverProjectID != "" && *stackdriverCreds != "" {
		close := stackdrivererror.Enable(ctx, *stackdriverProjectID, *stackdriverCreds, *stackdriverSamplePercentage, *externalIP)
		defer close()
	}

	go periodicallyForceGC()

	p := &proxy.Proxy{
		HTTPAddr:                  *addr,
		HTTPMultiplexAddr:         *multiplexAddr,
		CertFile:                  *certfile,
		CfgSvrAuthToken:           *cfgSvrAuthToken,
		CfgSvrDomains:             *cfgSvrDomains,
		ConnectOKWaitsForUpstream: *connectOKWaitsForUpstream,
		EnableReports:             *enableReports,
		ThrottleRefreshInterval:   *throttleRefreshInterval,
		ThrottleThreshold:         *throttleThreshold,
		ThrottleRate:              *throttleRate,
		BordaReportInterval:       *bordaReportInterval,
		BordaSamplePercentage:     *bordaSamplePercentage,
		BordaBufferSize:           *bordaBufferSize,
		ExternalIP:                *externalIP,
		HTTPS:                     *https,
		IdleTimeout:               time.Duration(*idleClose) * time.Second,
		KeyFile:                   *keyfile,
		Pro:                       *pro,
		ProxiedSitesSamplePercentage: *proxiedSitesSamplePercentage,
		ProxiedSitesTrackingID:       *proxiedSitesTrackingId,
		ReportingRedisAddr:           *reportingRedisAddr,
		ReportingRedisCA:             *reportingRedisCA,
		ReportingRedisClientPK:       *reportingRedisClientPK,
		ReportingRedisClientCert:     *reportingRedisClientCert,
		Token:                              *token,
		TunnelPorts:                        *tunnelPorts,
		Obfs4Addr:                          *obfs4Addr,
		Obfs4MultiplexAddr:                 *obfs4MultiplexAddr,
		Obfs4Dir:                           *obfs4Dir,
		Obfs4HandshakeConcurrency:          *obfs4HandshakeConcurrency,
		Obfs4MaxPendingHandshakesPerClient: *obfs4MaxPendingHandshakesPerClient,
		Obfs4HandshakeTimeout:              *obfs4HandshakeTimeout,
		KCPConf:                            *kcpConf,
		ENHTTPAddr:                         *enhttpAddr,
		ENHTTPServerURL:                    *enhttpServerURL,
		ENHTTPReapIdleTime:                 *enhttpReapIdleTime,
		Benchmark:                          *bench,
		FasttrackDomains:                   *fasttrackDomains,
		DiffServTOS:                        *tos,
		LampshadeAddr:                      *lampshadeAddr,
		VersionCheck:                       *versionCheck != "",
		VersionCheckRange:                  *versionCheck,
		VersionCheckRedirectURL:            *versionCheckRedirectURL,
		VersionCheckRedirectPercentage:     *versionCheckRedirectPercentage,
		GoogleSearchRegex:                  *googleSearchRegex,
		GoogleCaptchaRegex:                 *googleCaptchaRegex,
		BlacklistMaxIdleTime:               *blacklistMaxIdleTime,
		BlacklistMaxConnectInterval:        *blacklistMaxConnectInterval,
		BlacklistAllowedFailures:           *blacklistAllowedFailures,
		BlacklistExpiration:                *blacklistExpiration,
		ProxyName:                          *proxyName,
		BBRUpstreamProbeURL:                *bbrUpstreamProbeURL,
		QUICAddr:                           *quicAddr,
		PCAPDir:                            *pcapDir,
		PCAPIPs:                            *pcapIPs,
		PCAPSPerIP:                         *pcapsPerIP,
	}

	err := p.ListenAndServe()
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
