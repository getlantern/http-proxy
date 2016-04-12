package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern"
	"github.com/getlantern/http-proxy/logging"
	"github.com/vharitonsky/iniflags"
)

var (
	log = golog.LoggerFor("lantern-proxy")

	addr                         = flag.String("addr", ":8080", "Address to listen")
	certfile                     = flag.String("cert", "", "Certificate file name")
	cfgSvrAuthToken              = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	cfgSvrDomains                = flag.String("cfgsvrdomains", "", "Config-server domains on which to attach auth token, separated by comma")
	enablePro                    = flag.Bool("enablepro", false, "Enable Lantern Pro support")
	enableReports                = flag.Bool("enablereports", false, "Enable stats reporting")
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

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	s := &httpproxylantern.Server{
		Addr:                         *addr,
		CertFile:                     *certfile,
		CfgSvrAuthToken:              *cfgSvrAuthToken,
		CfgSvrDomains:                *cfgSvrDomains,
		EnablePro:                    *enablePro,
		EnableReports:                *enableReports,
		HTTPS:                        *https,
		IdleClose:                    *idleClose,
		Keyfile:                      *keyfile,
		MaxConns:                     *maxConns,
		ProxiedSitesSamplePercentage: *proxiedSitesSamplePercentage,
		ProxiedSitesTrackingId:       *proxiedSitesTrackingId,
		RedisAddr:                    *redisAddr,
		ServerId:                     *serverId,
		Token:                        *token,
		TunnelPorts:                  *tunnelPorts,
		Obfs4Addr:                    *obfs4Addr,
		Obfs4Dir:                     *obfs4Dir,
	}
	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
