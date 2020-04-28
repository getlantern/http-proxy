package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/mitchellh/panicwrap"
	"github.com/vharitonsky/iniflags"

	"github.com/getlantern/geo"
	"github.com/getlantern/golog"

	proxy "github.com/getlantern/http-proxy-lantern"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/googlefilter"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/stackdrivererror"
	"github.com/getlantern/http-proxy-lantern/throttle"
	"github.com/getlantern/http-proxy-lantern/tlslistener"
	"github.com/getlantern/quicwrapper"
)

var (
	log      = golog.LoggerFor("lantern-proxy")
	revision = "unknown" // overridden by Makefile
	// Use our own CDN distribution which fetches the origin at most once per
	// day to avoid hitting the 2000 downloads/day limit imposed by MaxMind.
	geolite2_url = "https://d254wvfcgkka1d.cloudfront.net/app/geoip_download?license_key=%s&edition_id=GeoLite2-Country&suffix=tar.gz"

	hostname, _ = os.Hostname()

	addr             = flag.String("addr", "", "Address to listen with HTTP(S)")
	multiplexAddr    = flag.String("multiplexaddr", "", "Multiplexed address at which to listen with HTTP(S)")
	utpAddr          = flag.String("utpaddr", "", "Address at which to listen with HTTP(S) over utp")
	lampshadeAddr    = flag.String("lampshade-addr", "", "Address at which to listen for lampshade connections with tcp. Requires https to be true.")
	lampshadeUTPAddr = flag.String("lampshade-utpaddr", "", "Address at which to listen for lampshade connections with utp. Requires https to be true.")
	quic0Addr        = flag.String("quic-addr", "", "Address at which to listen for legacy QUIC connections. (deprecated)")
	quicIETFAddr     = flag.String("quic-ietf-addr", "", "Address at which to listen for IETF QUIC connections.")
	wssAddr          = flag.String("wss-addr", "", "Address at which to listen for WSS connections.")
	kcpConf          = flag.String("kcpconf", "", "Path to file configuring kcp")

	obfs4Addr                          = flag.String("obfs4-addr", "", "Provide an address here in order to listen with obfs4")
	obfs4MultiplexAddr                 = flag.String("obfs4-multiplexaddr", "", "Provide an address here in order to listen with multiplexed obfs4")
	obfs4UTPAddr                       = flag.String("obfs4-utpaddr", "", "Provide an address here in order to listen with obfs4 over utp")
	obfs4Dir                           = flag.String("obfs4-dir", ".", "Directory where obfs4 can store its files")
	obfs4HandshakeConcurrency          = flag.Int("obfs4-handshake-concurrency", obfs4listener.DefaultHandshakeConcurrency, "How many concurrent OBFS4 handshakes to process")
	obfs4MaxPendingHandshakesPerClient = flag.Int("obfs4-max-pending-handshakes-per-client", obfs4listener.DefaultMaxPendingHandshakesPerClient, "How many pending OBFS4 handshakes to allow per client")
	obfs4HandshakeTimeout              = flag.Duration("obfs4-handshake-timeout", obfs4listener.DefaultHandshakeTimeout, "How long to wait before timing out an OBFS4 handshake")

	oquicDefaults          = quicwrapper.DefaultOQuicConfig([]byte(""))
	oquicAddr              = flag.String("oquic-addr", "", "Address at which to listen for OQUIC connections.")
	oquicCipher            = flag.String("oquic-cipher", oquicDefaults.Cipher, "OQUIC cipher")
	oquicKey               = flag.String("oquic-key", "", "OQUIC base64 encoded 256 bit obfuscation key")
	oquicAggressivePadding = flag.Uint64("oquic-aggressive-padding", uint64(oquicDefaults.AggressivePadding), "OQUIC number of initial aggressively padded packets")
	oquicMaxPaddingHint    = flag.Uint64("oquic-max-padding-hint", uint64(oquicDefaults.MaxPaddingHint), "OQUIC max padding after aggressive phase")
	oquicMinPadded         = flag.Uint64("oquic-min-padded", uint64(oquicDefaults.MinPadded), "OQUIC minimum size packet to pad")

	enhttpAddr         = flag.String("enhttp-addr", "", "Address at which to accept encapsulated HTTP requests")
	enhttpServerURL    = flag.String("enhttp-server-url", "", "specify a full URL for domain-fronting to this server with enhttp, required for sticky routing with CloudFront")
	enhttpReapIdleTime = flag.Duration("enhttp-reapidletime", time.Duration(*idleClose)*time.Second, "configure how long enhttp connections are allowed to remain idle before being forcibly closed")

	packetForwardAddr = flag.String("pforward-addr", "", "Address at which to listen for packet forwarding connections")
	packetForwardIntf = flag.String("pforward-intf", "eth0", "The name of the interface to use for upstream packet forwarding connections")

	keyfile              = flag.String("key", "", "Private key file name")
	certfile             = flag.String("cert", "", "Certificate file name")
	token                = flag.String("token", "", "Lantern token")
	sessionTicketKeyFile = flag.String("sessionticketkey", "", "File name for storing rotating session ticket keys")

	lampshadeKeyCacheSize     = flag.Int("lampshade-keycache-size", 0, "set this to a positive value to cache client keys and reject duplicates to thwart replay attacks")
	lampshadeMaxClientInitAge = flag.Duration("lampshade-max-clientinit-age", 0, "set this to a positive value to limit the age of client init messages to thwart replay attacks")

	cfgSvrAuthToken           = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	connectOKWaitsForUpstream = flag.Bool("connect-ok-waits-for-upstream", false, "Set to true to wait for upstream connection before responding OK to CONNECT requests")

	throttleRefreshInterval = flag.Duration("throttlerefresh", throttle.DefaultRefreshInterval, "Specifies how frequently to refresh throttling configuration from redis. Defaults to 5 minutes.")
	throttleThreshold       = flag.Int64("throttlethreshold", 0, "Set to a positive value to force a specific throttle threshold in bytes (rather than using one from Redis)")
	throttleRate            = flag.Int64("throttlerate", 0, "Set to a positive value to force a specific throttle rate in bytes/second (rather than using one from Redis)")

	enableReports         = flag.Bool("enablereports", false, "Enable stats reporting")
	bordaReportInterval   = flag.Duration("borda-report-interval", 0*time.Second, "How frequently to report errors to borda. Set to 0 to disable reporting.")
	bordaSamplePercentage = flag.Float64("borda-sample-percentage", 0.0001, "The percentage of devices to report to Borda (0.01 = 1%)")
	bordaBufferSize       = flag.Int("borda-buffer-size", 10000, "Size of borda buffer, caps how many distinct measurements to keep during each submit interval")

	externalIP = flag.String("externalip", "", "The external IP of this proxy, used for reporting to Borda")
	https      = flag.Bool("https", false, "Use TLS for client to proxy communication")
	idleClose  = flag.Uint64("idleclose", 70, "Time in seconds that an idle connection will be allowed before closing it")
	_          = flag.Uint64("maxconns", 0, "Max number of simultaneous allowed connections, unused")

	pprofAddr         = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	promExporterAddr  = flag.String("promexporteraddr", "", "Prometheus exporter address to listen on, not activate exporter if empty")
	maxmindLicenseKey = flag.String("maxmindlicensekey", "", "MaxMind license key to load the GeoLite2 Country database")
	geolite2DBFile    = flag.String("geolite2dbfile", "GeoLite2-Country.mmdb", "The local copy of the GeoLite2 Country database for bandwidth conservation and faster initialization")

	pro = flag.Bool("pro", false, "Set to true to make this a pro proxy (no bandwidth limiting unless forced throttling)")

	proxiedSitesSamplePercentage = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
	proxiedSitesTrackingId       = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")

	reportingRedisAddr       = flag.String("reportingredis", "", "The address of the reporting Redis instance in \"redis[s]://host:port\" format")
	reportingRedisCA         = flag.String("reportingredisca", "", "Certificate for the CA of Redis instance for reporting")
	reportingRedisClientPK   = flag.String("reportingredisclientpk", "", "Private key for authenticating client to the Redis instance for reporting")
	reportingRedisClientCert = flag.String("reportingredisclientcert", "", "Certificate for authenticating client to the Redis instance for reporting")

	tunnelPorts         = flag.String("tunnelports", "", "Comma seperated list of ports allowed for HTTP CONNECT tunnel. Allow all ports if empty.")
	tos                 = flag.Int("tos", 0, "Specify a diffserv TOS to prioritize traffic. Defaults to 0 (off)")
	proxyName           = flag.String("proxyname", hostname, "The name of this proxy (defaults to hostname)")
	proxyProtocol       = flag.String("proxyprotocol", "", "The protocol of this proxy, for information only")
	bbrUpstreamProbeURL = flag.String("bbrprobeurl", "", "optional URL to probe for upstream BBR bandwidth estimates")

	bench   = flag.Bool("bench", false, "Set this flag to set up proxy as a benchmarking proxy. This automatically puts the proxy into tls mode and disables auth token authentication.")
	version = flag.Bool("version", false, "shows the version of the binary")
	help    = flag.Bool("help", false, "Get usage help")

	versionCheck                   = flag.String("versioncheck", "", "Check if Lantern client matches the semantic version range, like \"< 3.1.1\" or \"<= 3.x\". No check by default")
	versionCheckRedirectURL        = flag.String("versioncheck-redirect-url", "", "The URL to redirect if client is below certain version. Always used along with versioncheck")
	versionCheckRedirectPercentage = flag.Float64("versioncheck-redirect-percentage", 1, "The percentage of requests to be redirected in version check. Defaults to 1 (100%)")

	googleSearchRegex  = flag.String("google-search-regex", googlefilter.DefaultSearchRegex, "Regex for detecting access to Google Search")
	googleCaptchaRegex = flag.String("google-captcha-regex", googlefilter.DefaultCaptchaRegex, "Regex for detecting access to Google captcha page")

	blacklistMaxIdleTime        = flag.Duration("blacklist-max-idle-time", blacklist.DefaultMaxIdleTime, "How long to wait for an HTTP request before considering a connection failed for blacklisting")
	blacklistMaxConnectInterval = flag.Duration("blacklist-max-connect-interval", blacklist.DefaultMaxConnectInterval, "Successive connection attempts within this interval will be treated as a single attempt for blacklisting")
	blacklistAllowedFailures    = flag.Int("blacklist-allowed-failures", blacklist.DefaultAllowedFailures, "The number of failed connection attempts we tolerate before blacklisting an IP address")
	blacklistExpiration         = flag.Duration("blacklist-expiration", blacklist.DefaultExpiration, "How long to wait before removing an ip from the blacklist")

	stackdriverProjectID        = flag.String("stackdriver-project-id", "lantern-http-proxy", "Optional project ID for stackdriver error reporting as in http-proxy-lantern")
	stackdriverCreds            = flag.String("stackdriver-creds", "/home/lantern/lantern-stackdriver.json", "Optional full json file path containing stackdriver credentials")
	stackdriverSamplePercentage = flag.Float64("stackdriver-sample-percentage", 0.003, "The percentage of devices to report to Stackdriver (0.01 = 1%)")

	pcapDir     = flag.String("pcap-dir", "/tmp", "Directory in which to save pcaps")
	pcapIPs     = flag.Int("pcap-ips", 0, "The number of IP addresses for which to capture packets")
	pcapsPerIP  = flag.Int("pcaps-per-ip", 0, "The number of packets to capture for each IP address")
	pcapSnapLen = flag.Int("pcap-snap-len", 1600, "The maximum size packet to capture")
	pcapTimeout = flag.Duration("pcap-timeout", 30*time.Millisecond, "Timeout for capturing packets")

	requireSessionTickets      = flag.Bool("require-session-tickets", true, "Specifies whether or not to require TLS session tickets in ClientHellos")
	missingTicketReaction      = flag.String("missing-session-ticket-reaction", "", "Specifies the reaction when seeing ClientHellos without TLS session tickets. Apply only if require-session-tickets is set")
	missingTicketReactionDelay = flag.Duration("missing-session-ticket-reaction-delay", 0, "Specifies the delay before reaction to ClientHellos without TLS session tickets. Apply only if require-session-tickets is set.")
	missingTicketReflectSite   = flag.String("missing-session-ticket-reflect-site", "", "Specifies the site to mirror when seeing no TLS session ticket in ClientHellos. Useful only if missing-session-ticket-reaction is ReflectToSite.")

	tlsmasqAddr          = flag.String("tlsmasq-addr", "", "Address at which to listen for tlsmasq connections.")
	tlsmasqOriginAddr    = flag.String("tlsmasq-origin-addr", "", "Address of tlsmasq origin with port.")
	tlsmasqSecret        = flag.String("tlsmasq-secret", "", "Hex encoded 52 byte tlsmasq shared secret.")
	tlsmasqMinVersionStr = flag.String("tlsmasq-tls-min-version", "0x0303", "hex-encoded TLS version")
	tlsmasqSuitesStr     = flag.String("tlsmasq-tls-cipher-suites", "0x1301,0x1302,0x1303,0xcca8,0xcca9,0xc02b,0xc030,0xc02c", "hex-encoded TLS cipher suites")
)

func main() {
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

	if *stackdriverProjectID != "" && *stackdriverCreds != "" {
		close := stackdrivererror.Enable(context.Background(), *stackdriverProjectID, *stackdriverCreds, *stackdriverSamplePercentage, *externalIP)
		defer close()
	}

	// panicwrap works by re-executing the running program (retaining arguments,
	// environmental variables, etc.) and monitoring the stderr of the program.
	exitStatus, panicWrapErr := panicwrap.Wrap(
		&panicwrap.WrapConfig{
			DetectDuration: time.Second,
			Handler: func(msg string) {
				log.Fatal(msg)
			},
			// Just forward signals to the child process
			ForwardSignals: []os.Signal{
				syscall.SIGHUP,
				syscall.SIGTERM,
				syscall.SIGQUIT,
				os.Interrupt,
			},
		})
	if panicWrapErr != nil {
		log.Fatalf("Error setting up panic wrapper: %v", panicWrapErr)
	} else {
		// If exitStatus >= 0, then we're the parent process.
		if exitStatus >= 0 {
			os.Exit(exitStatus)
		}
	}
	// We're in the child (wrapped) process now

	if (*lampshadeAddr != "" || *lampshadeUTPAddr != "") && !*https {
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
		log.Fatal("version check redirect URL should not be empty")
	}

	var reaction tlslistener.HandshakeReaction
	switch *missingTicketReaction {
	case "AlertHandshakeFailure":
		reaction = tlslistener.AlertHandshakeFailure
	case "AlertProtocolVersion":
		reaction = tlslistener.AlertProtocolVersion
	case "AlertInternalError":
		reaction = tlslistener.AlertInternalError
	case "CloseConnection":
		reaction = tlslistener.CloseConnection
	case "ReflectToSite":
		if *missingTicketReflectSite == "" {
			log.Fatal("missing-session-ticket-reflect-site should not be empty")
		}
		reaction = tlslistener.ReflectToSite(*missingTicketReflectSite)
		log.Debugf("Reflecting missing session tickets to site %v", *missingTicketReflectSite)
	default:
		reaction = tlslistener.AlertInternalError
		log.Errorf("unrecognized missing-session-ticket-reaction %s, fallback to %s", *missingTicketReaction, reaction.Action())
	}
	if *missingTicketReactionDelay != 0 {
		reaction = tlslistener.Delayed(*missingTicketReactionDelay, reaction)
	}

	if reaction.Action() == "" {
		log.Debug("Not using missing-session-ticket-reaction")
	} else {
		log.Debugf("Using missing-session-ticket-reaction %v", reaction.Action())
	}

	var (
		tlsmasqTLSMinVersion uint16
		tlsmasqTLSSuites     []uint16
		err                  error
	)
	if *tlsmasqAddr != "" && (*tlsmasqSecret == "" || *tlsmasqOriginAddr == "") {
		log.Fatalf("tlsmasq requires tlsmasq-secret and tlsmasq-origin-addr")
	}
	if *tlsmasqMinVersionStr != "" {
		tlsmasqTLSMinVersion, err = decodeUint16(*tlsmasqMinVersionStr)
		if err != nil {
			log.Fatal(fmt.Sprintln("failed to decode tlsmasq-tls-min-version:", err))
		}
	}
	if *tlsmasqSuitesStr != "" {
		tlsmasqTLSSuites = []uint16{}
		for _, s := range strings.Split(*tlsmasqSuitesStr, ",") {
			suite, err := decodeUint16(s)
			if err != nil {
				log.Fatal(fmt.Sprintln("failed to decode tlsmasq-tls-cipher-suites:", err))
			}
			tlsmasqTLSSuites = append(tlsmasqTLSSuites, suite)
		}
	}
	var geoLookup geo.Lookup
	if *maxmindLicenseKey == "" {
		log.Fatal("maxmindlicensekey should not be empty")
	}
	geoLookup = geo.New(fmt.Sprintf(geolite2_url, *maxmindLicenseKey), 24*time.Hour, *geolite2DBFile)

	go periodicallyForceGC()

	p := &proxy.Proxy{
		HTTPAddr:                           *addr,
		HTTPMultiplexAddr:                  *multiplexAddr,
		HTTPUTPAddr:                        *utpAddr,
		CertFile:                           *certfile,
		CfgSvrAuthToken:                    *cfgSvrAuthToken,
		ConnectOKWaitsForUpstream:          *connectOKWaitsForUpstream,
		EnableReports:                      *enableReports,
		ThrottleRefreshInterval:            *throttleRefreshInterval,
		ThrottleThreshold:                  *throttleThreshold,
		ThrottleRate:                       *throttleRate,
		BordaReportInterval:                *bordaReportInterval,
		BordaSamplePercentage:              *bordaSamplePercentage,
		BordaBufferSize:                    *bordaBufferSize,
		ExternalIP:                         *externalIP,
		HTTPS:                              *https,
		IdleTimeout:                        time.Duration(*idleClose) * time.Second,
		KeyFile:                            *keyfile,
		SessionTicketKeyFile:               *sessionTicketKeyFile,
		Pro:                                *pro,
		ProxiedSitesSamplePercentage:       *proxiedSitesSamplePercentage,
		ProxiedSitesTrackingID:             *proxiedSitesTrackingId,
		ReportingRedisAddr:                 *reportingRedisAddr,
		ReportingRedisCA:                   *reportingRedisCA,
		ReportingRedisClientPK:             *reportingRedisClientPK,
		ReportingRedisClientCert:           *reportingRedisClientCert,
		Token:                              *token,
		TunnelPorts:                        *tunnelPorts,
		Obfs4Addr:                          *obfs4Addr,
		Obfs4MultiplexAddr:                 *obfs4MultiplexAddr,
		Obfs4UTPAddr:                       *obfs4UTPAddr,
		Obfs4Dir:                           *obfs4Dir,
		Obfs4HandshakeConcurrency:          *obfs4HandshakeConcurrency,
		Obfs4MaxPendingHandshakesPerClient: *obfs4MaxPendingHandshakesPerClient,
		Obfs4HandshakeTimeout:              *obfs4HandshakeTimeout,
		OQUICAddr:                          *oquicAddr,
		OQUICKey:                           *oquicKey,
		OQUICCipher:                        *oquicCipher,
		OQUICAggressivePadding:             *oquicAggressivePadding,
		OQUICMaxPaddingHint:                *oquicMaxPaddingHint,
		OQUICMinPadded:                     *oquicMinPadded,
		KCPConf:                            *kcpConf,
		ENHTTPAddr:                         *enhttpAddr,
		ENHTTPServerURL:                    *enhttpServerURL,
		ENHTTPReapIdleTime:                 *enhttpReapIdleTime,
		Benchmark:                          *bench,
		DiffServTOS:                        *tos,
		LampshadeAddr:                      *lampshadeAddr,
		LampshadeUTPAddr:                   *lampshadeUTPAddr,
		LampshadeKeyCacheSize:              *lampshadeKeyCacheSize,
		LampshadeMaxClientInitAge:          *lampshadeMaxClientInitAge,
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
		ProxyProtocol:                      *proxyProtocol,
		BBRUpstreamProbeURL:                *bbrUpstreamProbeURL,
		QUICIETFAddr:                       *quicIETFAddr,
		QUIC0Addr:                          *quic0Addr,
		WSSAddr:                            *wssAddr,
		PCAPDir:                            *pcapDir,
		PCAPIPs:                            *pcapIPs,
		PCAPSPerIP:                         *pcapsPerIP,
		PCAPSnapLen:                        *pcapSnapLen,
		PCAPTimeout:                        *pcapTimeout,
		PacketForwardAddr:                  *packetForwardAddr,
		PacketForwardIntf:                  *packetForwardIntf,
		RequireSessionTickets:              *requireSessionTickets,
		MissingTicketReaction:              reaction,
		TLSMasqAddr:                        *tlsmasqAddr,
		TLSMasqOriginAddr:                  *tlsmasqOriginAddr,
		TLSMasqSecret:                      *tlsmasqSecret,
		TLSMasqTLSMinVersion:               tlsmasqTLSMinVersion,
		TLSMasqTLSCipherSuites:             tlsmasqTLSSuites,
		PromExporterAddr:                   *promExporterAddr,
		GeoLookup:                          geoLookup,
	}

	log.Fatal(p.ListenAndServe())
}

func periodicallyForceGC() {
	for {
		time.Sleep(1 * time.Minute)
		debug.FreeOSMemory()
	}
}

func decodeUint16(s string) (uint16, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}
