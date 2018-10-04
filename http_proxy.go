package proxy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/enhttp"
	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/kcpwrapper"
	"github.com/getlantern/ops"
	"github.com/getlantern/proxy"
	"github.com/getlantern/proxy/filters"
	"github.com/getlantern/quicwrapper"
	"github.com/getlantern/tlsdefaults"
	"github.com/getlantern/tlsredis"
	rclient "gopkg.in/redis.v5"

	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/proxyfilters"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/bbr"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/borda"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	"github.com/getlantern/http-proxy-lantern/diffserv"
	"github.com/getlantern/http-proxy-lantern/googlefilter"
	"github.com/getlantern/http-proxy-lantern/lampshade"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/opsfilter"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/quic"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/throttle"
	"github.com/getlantern/http-proxy-lantern/tlslistener"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
	"github.com/getlantern/http-proxy-lantern/versioncheck"
)

const (
	timeoutToDialOriginSite = 10 * time.Second
)

var (
	log = golog.LoggerFor("lantern-proxy")

	proxyNameRegex = regexp.MustCompile(`(fp-([a-z0-9]+-)?([a-z0-9]+)-[0-9]{8}-[0-9]+)(-.+)?`)
)

// Proxy is an HTTP proxy.
type Proxy struct {
	TestingLocal                       bool
	Addr                               string
	BordaReportInterval                time.Duration
	BordaSamplePercentage              float64
	BordaBufferSize                    int
	ExternalIP                         string
	CertFile                           string
	CfgSvrAuthToken                    string
	CfgSvrDomains                      string
	CfgSvrCacheClear                   time.Duration
	ConnectOKWaitsForUpstream          bool
	ENHTTPAddr                         string
	ENHTTPServerURL                    string
	ENHTTPReapIdleTime                 time.Duration
	EnableReports                      bool
	HTTPS                              bool
	IdleTimeout                        time.Duration
	KeyFile                            string
	Pro                                bool
	ProxiedSitesSamplePercentage       float64
	ProxiedSitesTrackingID             string
	ReportingRedisAddr                 string
	ReportingRedisCA                   string
	ReportingRedisClientPK             string
	ReportingRedisClientCert           string
	ThrottleRefreshInterval            time.Duration
	ThrottleThreshold                  int64
	ThrottleRate                       int64
	Token                              string
	TunnelPorts                        string
	Obfs4Addr                          string
	Obfs4Dir                           string
	Obfs4HandshakeConcurrency          int
	Obfs4MaxPendingHandshakesPerClient int
	Obfs4HandshakeTimeout              time.Duration
	KCPConf                            string
	Benchmark                          bool
	FasttrackDomains                   string
	DiffServTOS                        int
	LampshadeAddr                      string
	VersionCheck                       bool
	VersionCheckRange                  string
	VersionCheckRedirectURL            string
	VersionCheckRedirectPercentage     float64
	GoogleSearchRegex                  string
	GoogleCaptchaRegex                 string
	BlacklistMaxIdleTime               time.Duration
	BlacklistMaxConnectInterval        time.Duration
	BlacklistAllowedFailures           int
	BlacklistExpiration                time.Duration
	ProxyName                          string
	BBRUpstreamProbeURL                string
	QUICAddr                           string

	bm             bbr.Middleware
	rc             *rclient.Client
	throttleConfig throttle.Config
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	p.setupOpsContext()
	p.setBenchmarkMode()
	p.bm = bbr.New()
	if p.BBRUpstreamProbeURL != "" {
		go p.bm.ProbeUpstream(p.BBRUpstreamProbeURL)
	}
	p.initRedisClient()
	p.loadThrottleConfig()

	// Only allow connections from remote IPs that are not blacklisted
	blacklist := p.createBlacklist()
	filterChain, dial, err := p.createFilterChain(blacklist)
	if err != nil {
		return err
	}

	if p.QUICAddr != "" {
		filterChain = filterChain.Prepend(quic.NewMiddleware())
	}

	bwReporting, bordaReporter := p.configureBandwidthReporting()
	srv := server.New(&server.Opts{
		IdleTimeout: p.IdleTimeout,
		Dial:        dial,
		Filter:      filterChain.Prepend(opsfilter.New(p.bm)),
		OKDoesNotWaitForUpstream: !p.ConnectOKWaitsForUpstream,
	})
	// Temporarily disable blacklisting
	// srv.Allow = blacklist.OnConnect
	p.applyThrottling(srv, bwReporting)
	srv.AddListenerWrappers(bwReporting.wrapper)
	if p.Obfs4Addr != "" {
		p.serveOBFS4(srv)
	}

	var l net.Listener
	if p.KCPConf != "" {
		l, err = p.listenKCP()
		if err != nil {
			return errors.New("Unable to listen kcp: %v", err)
		}
	} else if p.QUICAddr != "" {
		l, err = p.listenQUIC()
	} else {
		l, err = p.listenTCP(p.Addr, true)
		if err != nil {
			return errors.New("Unable to listen tcp at %s: %v", p.Addr, err)
		}
	}

	protocol := "HTTP"
	if p.HTTPS {
		protocol = "HTTPS"
		l, err = tlslistener.Wrap(l, p.KeyFile, p.CertFile)
		if err != nil {
			return err
		}
		// We initialize lampshade here because it uses the same keypair as HTTPS
		if p.LampshadeAddr != "" {
			p.serveLampshade(srv, bordaReporter)
		}
	}

	log.Debugf("Listening for %v at %v", protocol, l.Addr())
	log.Debugf("Type of listener: %v", reflect.TypeOf(l))

	if p.ENHTTPAddr != "" {
		el, err := net.Listen("tcp", p.ENHTTPAddr)
		if err != nil {
			return errors.New("Unable to listen for encapsulated HTTP at %v: %v", p.ENHTTPAddr, err)
		}
		log.Debugf("Listening for encapsulated HTTP at %v", el.Addr())
		filterChain := filters.Join(tokenfilter.New(p.Token), ping.New(0))
		enhttpHandler := enhttp.NewServerHandler(p.ENHTTPReapIdleTime, p.ENHTTPServerURL)
		server := &http.Server{
			Handler: filters.Intercept(enhttpHandler, filterChain),
		}
		return server.Serve(el)
	}

	err = srv.Serve(l, mimic.SetServerAddr)
	if err != nil {
		return errors.New("Error serving HTTP(S): %v", err)
	}

	return nil
}

func (p *Proxy) setupOpsContext() {
	ops.SetGlobal("app", "http-proxy")
	if p.ExternalIP != "" {
		log.Debugf("Will report with proxy_host: %v", p.ExternalIP)
		ops.SetGlobal("proxy_host", p.ExternalIP)
	}
	proxyName, dc := proxyName(p.ProxyName)
	// Only set proxy name if it follows our naming convention
	if proxyName != "" {
		log.Debugf("Will report with proxy_name %v in dc %v", proxyName, dc)
		ops.SetGlobal("proxy_name", proxyName)
		ops.SetGlobal("dc", dc)
	}
	ops.SetGlobal("proxy_protocol", p.proxyProtocol())
	ops.SetGlobal("is_pro", p.Pro)
}

func proxyName(hostname string) (proxyName string, dc string) {
	match := proxyNameRegex.FindStringSubmatch(hostname)
	// Only set proxy name if it follows our naming convention
	if len(match) != 5 {
		return "", ""
	}
	return match[1], match[3]
}

func (p *Proxy) proxyProtocol() string {
	if p.LampshadeAddr != "" {
		return "lampshade"
	}
	if p.KCPConf != "" {
		return "kcp"
	}
	if p.QUICAddr != "" {
		return "quic"
	}
	return "https"
}

func (p *Proxy) setBenchmarkMode() {
	if p.Benchmark {
		log.Debug("Putting proxy into benchmarking mode. Only a limited rate of requests to a specific set of domains will be allowed, no authentication token required.")
		p.HTTPS = true
		p.Token = "bench"
	}
}

func (p *Proxy) createBlacklist() *blacklist.Blacklist {
	return blacklist.New(blacklist.Options{
		MaxIdleTime:        p.BlacklistMaxIdleTime,        // 30 * time.Second,
		MaxConnectInterval: p.BlacklistMaxConnectInterval, // 5 * time.Second,
		AllowedFailures:    p.BlacklistAllowedFailures,    // 10,
		Expiration:         p.BlacklistExpiration,         // 6 * time.Hour,
	})
}

// createFilterChain creates a chain of filters that modify the default behavior
// of proxy.Proxy to implement Lantern-specific logic like authentication,
// Apache mimicry, bandwidth throttling, BBR metric reporting, etc. The actual
// work of proxying plain HTTP and CONNECT requests is handled by proxy.Proxy
// itself.
func (p *Proxy) createFilterChain(bl *blacklist.Blacklist) (filters.Chain, proxy.DialFunc, error) {
	filterChain := filters.Join(p.bm)

	if p.Benchmark {
		filterChain = filterChain.Append(proxyfilters.RateLimit(5000, map[string]time.Duration{
			"www.google.com":      30 * time.Minute,
			"www.facebook.com":    30 * time.Minute,
			"67.media.tumblr.com": 30 * time.Minute,
			"i.ytimg.com":         30 * time.Minute,    // YouTube play button
			"149.154.167.91":      30 * time.Minute,    // Telegram
			"ping-chained-server": 1 * time.Nanosecond, // Internal ping-chained-server protocol
		}))
	} else {
		filterChain = filterChain.Append(proxy.OnFirstOnly(tokenfilter.New(p.Token)))
	}

	fd := common.NewRawFasttrackDomains(p.FasttrackDomains)
	if p.rc == nil {
		log.Debug("Not enabling bandwidth limiting")
	} else {
		filterChain = filterChain.Append(
			proxy.OnFirstOnly(devicefilter.NewPre(
				redis.NewDeviceFetcher(p.rc), p.throttleConfig, fd, !p.Pro)),
		)
	}

	filterChain = filterChain.Append(
		proxy.OnFirstOnly(googlefilter.New(p.GoogleSearchRegex, p.GoogleCaptchaRegex)),
		analytics.New(&analytics.Options{
			TrackingID:       p.ProxiedSitesTrackingID,
			SamplePercentage: p.ProxiedSitesSamplePercentage,
		}),
		proxy.OnFirstOnly(devicefilter.NewPost(bl)),
	)

	if !p.TestingLocal {
		filterChain = filterChain.Append(proxyfilters.BlockLocal([]string{"127.0.0.1:7300"}))
	}
	filterChain = filterChain.Append(ping.New(0))

	// Google anomaly detection can be triggered very often over IPv6.
	// Prefer IPv4 to mitigate, see issue #97
	dialer := preferIPV4Dialer(timeoutToDialOriginSite)
	dialerForPforward := dialer

	var requestRewriters []func(*http.Request)
	if p.CfgSvrAuthToken != "" || p.CfgSvrDomains != "" {
		cfg := &configserverfilter.Options{
			AuthToken: p.CfgSvrAuthToken,
			Domains:   strings.Split(p.CfgSvrDomains, ","),
		}
		dialerForPforward = configserverfilter.Dialer(dialerForPforward, cfg)
		csf := configserverfilter.New(cfg)
		filterChain = filterChain.Append(csf)
		requestRewriters = append(requestRewriters, csf.RewriteIfNecessary)
	}

	// Check if Lantern client version is in the supplied range. If yes,
	// redirect certain percentage of the requests to an URL to notify the user
	// to upgrade.
	if p.VersionCheck {
		log.Debugf("versioncheck: Will redirect %.4f%% of requests from Lantern clients below %s to %s",
			p.VersionCheckRedirectPercentage*100,
			p.VersionCheckRange,
			p.VersionCheckRedirectURL,
		)
		vc, err := versioncheck.New(p.VersionCheckRange,
			p.VersionCheckRedirectURL,
			[]string{"80"}, // checks CONNECT tunnel to 80 port only.
			p.VersionCheckRedirectPercentage)
		if err != nil {
			log.Errorf("Fail to init versioncheck, skipping: %v", err)
		} else {
			dialerForPforward = vc.Dialer(dialerForPforward)
			filterChain = filterChain.Append(vc.Filter())
		}
	}

	if len(requestRewriters) > 0 {
		filterChain = filterChain.Append(filters.FilterFunc(func(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
			if req.Method != http.MethodConnect {
				for _, rw := range requestRewriters {
					rw(req)
				}
			}
			return next(ctx, req)
		}))
	}

	filterChain = filterChain.Append(
		proxyfilters.DiscardInitialPersistentRequest,
		filters.FilterFunc(func(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
			if fd.Whitelisted(req) {
				// Only add X-Forwarded-For for our fasttrack domains
				return proxyfilters.AddForwardedFor(ctx, req, next)
			}
			return next(ctx, req)
		}),
		proxyfilters.RestrictConnectPorts(p.allowedTunnelPorts()),
		proxyfilters.RecordOp,
	)

	return filterChain, func(ctx context.Context, isCONNECT bool, network, addr string) (net.Conn, error) {
		if isCONNECT {
			return dialer(ctx, network, addr)
		}
		return dialerForPforward(ctx, network, addr)
	}, nil
}

func (p *Proxy) configureBandwidthReporting() (*reportingConfig, listeners.MeasuredReportFN) {
	var bordaReporter listeners.MeasuredReportFN
	if p.BordaReportInterval > 0 {
		bordaReporter = borda.Enable(p.BordaReportInterval, p.BordaSamplePercentage, p.BordaBufferSize)
	}
	return newReportingConfig(p.rc, p.EnableReports, bordaReporter), bordaReporter
}

func (p *Proxy) loadThrottleConfig() {
	if p.ThrottleThreshold > 0 && p.ThrottleRate > 0 {
		log.Debugf("Forcing throttling threshold and rate to %d : %d",
			p.ThrottleThreshold,
			p.ThrottleRate)
		p.throttleConfig = throttle.NewForcedConfig(p.ThrottleThreshold, p.ThrottleRate)
	} else if !p.Pro && p.ThrottleRefreshInterval > 0 && p.rc != nil {
		p.throttleConfig = throttle.NewRedisConfig(p.rc, p.ThrottleRefreshInterval)
	} else {
		log.Debug("Not loading throttle config")
		return
	}
}

func (p *Proxy) applyThrottling(srv *server.Server, rc *reportingConfig) {
	if p.throttleConfig == nil {
		log.Debug("Throttling is disabled")
	}

	// Add net.Listener wrappers for inbound connections
	srv.AddListenerWrappers(
		// Throttle connections when signaled
		func(ls net.Listener) net.Listener {
			return lanternlisteners.NewBitrateListener(ls)
		},
	)
}

func (p *Proxy) allowedTunnelPorts() []int {
	if p.TunnelPorts == "" {
		log.Debug("tunnelling all ports")
		return nil
	}
	ports, err := portsFromCSV(p.TunnelPorts)
	if err != nil {
		log.Fatal(err)
	}
	return ports
}

func (p *Proxy) initRedisClient() {
	var err error
	if p.ReportingRedisAddr == "" {
		log.Debug("no redis address configured for bandwidth reporting")
		return
	}

	redisOpts := &tlsredis.Options{
		RedisURL:       p.ReportingRedisAddr,
		RedisCAFile:    p.ReportingRedisCA,
		ClientPKFile:   p.ReportingRedisClientPK,
		ClientCertFile: p.ReportingRedisClientCert,
	}
	p.rc, err = tlsredis.GetClient(redisOpts)
	if err != nil {
		log.Errorf("Error connecting to redis, will not be able to perform bandwidth limiting: %v", err)
	}
}

func (p *Proxy) serveOBFS4(srv *server.Server) {
	l, listenErr := p.listenTCP(p.Obfs4Addr, true)
	if listenErr != nil {
		log.Fatalf("Unable to listen for OBFS4 with tcp: %v", listenErr)
	}
	wrapped, wrapErr := obfs4listener.Wrap(l, p.Obfs4Dir, p.Obfs4HandshakeConcurrency, p.Obfs4MaxPendingHandshakesPerClient, p.Obfs4HandshakeTimeout)
	if wrapErr != nil {
		log.Fatalf("Unable to listen with obfs4 at %v: %v", l.Addr(), wrapErr)
	}
	log.Debugf("Listening for OBFS4 at %v", wrapped.Addr())

	go func() {
		serveErr := srv.Serve(wrapped, func(addr string) {
			log.Debugf("obfs4 serving at %v", addr)
		})
		if serveErr != nil {
			log.Fatalf("Error serving obfs4 at %v: %v", wrapped.Addr(), serveErr)
		}
	}()
}

func (p *Proxy) serveLampshade(srv *server.Server, bordaReporter listeners.MeasuredReportFN) {
	l, err := p.listenTCP(p.LampshadeAddr, false)
	if err != nil {
		log.Fatalf("Unable to listen for lampshade with tcp: %v", err)
	}
	if bordaReporter != nil {
		log.Debug("Wrapping lampshade's TCP listener with measured reporting")
		l = listeners.NewMeasuredListener(l,
			measuredReportingInterval,
			borda.ConnectionTypedBordaReporter("physical", bordaReporter))
	}
	wrapped, wrapErr := lampshade.Wrap(l, p.CertFile, p.KeyFile)
	if wrapErr != nil {
		log.Fatalf("Unable to initialize lampshade with tcp: %v", wrapErr)
	}
	log.Debugf("Listening for lampshade at %v", wrapped.Addr())

	// We wrap the lampshade listener itself so that we record BBR metrics on
	// close of virtual streams rather than the physical connection.
	wrapped = p.bm.Wrap(wrapped)
	go func() {
		serveErr := srv.Serve(wrapped, func(addr string) {
			log.Debugf("lampshade serving at %v", addr)
		})
		if serveErr != nil {
			log.Fatalf("Error serving lampshade at %v: %v", wrapped.Addr(), serveErr)
		}
	}()
}

func (p *Proxy) listenTCP(addr string, wrapBBR bool) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if p.IdleTimeout > 0 {
		l = listeners.NewIdleConnListener(l, p.IdleTimeout)
	}
	if p.DiffServTOS > 0 {
		log.Debugf("Setting diffserv TOS to %d", p.DiffServTOS)
		// Note - this doesn't actually wrap the underlying connection, it'll still
		// be a net.TCPConn
		l = diffserv.Wrap(l, p.DiffServTOS)
	} else {
		log.Debugf("Not setting diffserv TOS")
	}
	if wrapBBR {
		l = p.bm.Wrap(l)
	}
	return l, nil
}

func (p *Proxy) listenKCP() (net.Listener, error) {
	cfg := &kcpwrapper.ListenerConfig{}
	file, err := os.Open(p.KCPConf) // For read access.
	if err != nil {
		return nil, errors.New("Unable to open KCPConf at %v: %v", p.KCPConf, err)
	}

	err = json.NewDecoder(file).Decode(cfg)
	file.Close()
	if err != nil {
		return nil, errors.New("Unable to decode KCPConf at %v: %v", p.KCPConf, err)
	}

	return kcpwrapper.Listen(cfg, func(conn net.Conn) net.Conn {
		if p.IdleTimeout <= 0 {
			return conn
		}
		return listeners.WrapIdleConn(conn, p.IdleTimeout)
	})
}

func (p *Proxy) listenQUIC() (net.Listener, error) {
	tlsConf, err := tlsdefaults.BuildListenerConfig(p.QUICAddr, p.KeyFile, p.CertFile)
	if err != nil {
		return nil, err
	}

	config := &quicwrapper.Config{
		MaxIncomingStreams: 1000,
	}

	return quicwrapper.ListenAddr(p.QUICAddr, tlsConf, config)
}

func portsFromCSV(csv string) ([]int, error) {
	fields := strings.Split(csv, ",")
	ports := make([]int, len(fields))
	for i, f := range fields {
		p, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return nil, err
		}
		ports[i] = p
	}
	return ports, nil
}
