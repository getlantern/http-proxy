package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/vharitonsky/iniflags"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/logging"

	"github.com/getlantern/http-proxy-lantern"
)

var (
	log = golog.LoggerFor("lantern-proxy")

	hostname, hostnameErr = os.Hostname()

	addr                         = flag.String("addr", ":8080", "Address to listen")
	certfile                     = flag.String("cert", "", "Certificate file name")
	cfgSvrAuthToken              = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	cfgSvrDomains                = flag.String("cfgsvrdomains", "", "Config-server domains on which to attach auth token, separated by comma")
	enablePro                    = flag.Bool("enablepro", false, "Enable Lantern Pro support")
	enableReports                = flag.Bool("enablereports", false, "Enable stats reporting")
	throttlebps                  = flag.Uint64("throttlebps", 0, "If > 0, enables throttling at the given bps (needs stats reporting enabled)")
	throttlethreshold            = flag.Uint64("throttlethreshold", 0, "If > 0, throttling will be activated at the given threshold (in bytes) in all connections of the throttled device")
	bordaReportInterval          = flag.Duration("borda-report-interval", 0*time.Second, "How frequently to report errors to borda. Set to 0 to disable reporting.")
	bordaSamplePercentage        = flag.Float64("borda-sample-percentage", 0.0001, "The percentage of devices to report to Borda (0.01 = 1%)")
	help                         = flag.Bool("help", false, "Get usage help")
	https                        = flag.Bool("https", false, "Use TLS for client to proxy communication")
	idleClose                    = flag.Uint64("idleclose", 70, "Time in seconds that an idle connection will be allowed before closing it")
	keyfile                      = flag.String("key", "", "Private key file name")
	logglyToken                  = flag.String("logglytoken", "", "Token used to report to loggly.com, not reporting if empty")
	maxConns                     = flag.Uint64("maxconns", 0, "Max number of simultaneous connections allowed connections")
	pprofAddr                    = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	proxiedSitesSamplePercentage = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
	proxiedSitesTrackingId       = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")
	redisAddr                    = flag.String("redis", "redis://localhost:6379", "Redis address in \"host:port\" format")
	redisCA                      = flag.String("redisca", "", "Certificate for redislabs's CA")
	redisClientPK                = flag.String("redisclientpk", "", "Private key for authenticating client to redis's stunnel")
	redisClientCert              = flag.String("redisclientcert", "", "Certificate for authenticating client to redis's stunnel")
	serverId                     = flag.String("serverid", hostname, "Server Id required for Pro-supporting servers")
	token                        = flag.String("token", "", "Lantern token")
	tunnelPorts                  = flag.String("tunnelports", "", "Comma seperated list of ports allowed for HTTP CONNECT tunnel. Allow all ports if empty.")
	obfs4Addr                    = flag.String("obfs4-addr", "", "Provide an address here in order to listen with obfs4")
	obfs4KCPAddr                 = flag.String("obfs4-kcp-addr", "", "Provide an address here in order to listen with obfs4 over KCP (udp-based TCP accelerator)")
	obfs4Dir                     = flag.String("obfs4-dir", ".", "Directory where obfs4 can store its files")
	bench                        = flag.Bool("bench", false, "Set this flag to set up proxy as a benchmarking proxy. This automatically puts the proxy into tls mode and disables auth token authentication.")
	whitelistDomains             = flag.String("whitelistdomains", "", "Whitelisted domains, such as the config server, pro server, etc, that should not count towards the bandwidth cap or be throttled, separated by comma")
)

func init() {
	if hostnameErr != nil {
		log.Errorf("Unable to obtain hostname: %v", hostnameErr)
	}
}

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

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	p := &proxy.Proxy{
		Addr:                  *addr,
		CertFile:              *certfile,
		CfgSvrAuthToken:       *cfgSvrAuthToken,
		CfgSvrDomains:         *cfgSvrDomains,
		EnablePro:             *enablePro,
		EnableReports:         *enableReports,
		ThrottleBPS:           *throttlebps,
		ThrottleThreshold:     *throttlethreshold,
		BordaReportInterval:   *bordaReportInterval,
		BordaSamplePercentage: *bordaSamplePercentage,
		HTTPS:                        *https,
		IdleClose:                    *idleClose,
		KeyFile:                      *keyfile,
		MaxConns:                     *maxConns,
		ProxiedSitesSamplePercentage: *proxiedSitesSamplePercentage,
		ProxiedSitesTrackingID:       *proxiedSitesTrackingId,
		RedisAddr:                    *redisAddr,
		RedisCA:                      *redisCA,
		RedisClientPK:                *redisClientPK,
		RedisClientCert:              *redisClientCert,
		ServerID:                     *serverId,
		Token:                        *token,
		TunnelPorts:                  *tunnelPorts,
		Obfs4Addr:                    *obfs4Addr,
		Obfs4KCPAddr:                 *obfs4KCPAddr,
		Obfs4Dir:                     *obfs4Dir,
		Benchmark:                    *bench,
		WhitelistDomains:             *whitelistDomains,
	}

	p.ListenAndServe()
}
