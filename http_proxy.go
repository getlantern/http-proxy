package proxy

import (
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy/commonfilter"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/http-proxy/forward"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/pforward"
	"github.com/getlantern/http-proxy/ratelimiter"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/bbr"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/borda"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	"github.com/getlantern/http-proxy-lantern/diffserv"
	"github.com/getlantern/http-proxy-lantern/lampshade"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/opsfilter"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/profilter"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/tlslistener"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
)

const (
	timeoutToDialOriginSite = 10 * time.Second
)

var (
	log = golog.LoggerFor("lantern-proxy")
)

// Proxy is an HTTP proxy.
type Proxy struct {
	TestingLocal                 bool
	Addr                         string
	BordaReportInterval          time.Duration
	BordaSamplePercentage        float64
	BordaBufferSize              int
	ExternalIP                   string
	CertFile                     string
	CfgSvrAuthToken              string
	CfgSvrDomains                string
	EnableReports                bool
	HTTPS                        bool
	IdleTimeout                  time.Duration
	KeyFile                      string
	ProxiedSitesSamplePercentage float64
	ProxiedSitesTrackingID       string
	RedisAddr                    string
	RedisCA                      string
	RedisClientPK                string
	RedisClientCert              string
	ServerID                     string
	ThrottleBPS                  uint64
	ThrottleThreshold            uint64
	Token                        string
	TunnelPorts                  string
	Obfs4Addr                    string
	Obfs4Dir                     string
	Benchmark                    bool
	FasttrackDomains             string
	DiffServTOS                  int
	LampshadeAddr                string
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	p.setupOpsContext()
	p.setBenchmarkMode()

	bbrEnabled := runtime.GOOS == "linux"
	// Only allow connections from remote IPs that are not blacklisted
	blacklist := p.createBlacklist()
	filterChain, err := p.createFilterChain(blacklist, bbrEnabled)
	if err != nil {
		return err
	}

	bwReporting := p.configureBandwidthReporting()
	srv := server.NewServer(filterChain.Prepend(opsfilter.New()))
	srv.Allow = blacklist.OnConnect
	if err := p.applyThrottling(srv, bwReporting); err != nil {
		return err
	}
	srv.AddListenerWrappers(bwReporting.wrapper)
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, p.IdleTimeout)
		},
	)

	if p.Obfs4Addr != "" {
		p.serveOBFS4(srv, bbrEnabled)
	}

	l, err := p.listenTCP(p.Addr, bbrEnabled)
	if err != nil {
		return errors.New("Unable to listen tcp at %s: %v", p.Addr, err)
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
			p.serveLampshade(srv, bbrEnabled)
		}
	}

	log.Debugf("Listening for %v at %v", protocol, l.Addr())
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
		MaxIdleTime:        30 * time.Second,
		MaxConnectInterval: 5 * time.Second,
		AllowedFailures:    10,
		Expiration:         6 * time.Hour,
	})
}

func (p *Proxy) createFilterChain(
	bl *blacklist.Blacklist,
	bbrEnabled bool,
) (filters.Chain, error) {
	var filterChain filters.Chain
	if bbrEnabled {
		log.Debug("Tracking bbr metrics")
		filterChain = filterChain.Append(bbr.NewFilter())
	} else {
		log.Debugf("OS is %v, not tracking bbr metrics", runtime.GOOS)
	}

	if p.Benchmark {
		filterChain = filterChain.Append(ratelimiter.New(5000, map[string]time.Duration{
			"www.google.com":      30 * time.Minute,
			"www.facebook.com":    30 * time.Minute,
			"67.media.tumblr.com": 30 * time.Minute,
			"i.ytimg.com":         30 * time.Minute,    // YouTube play button
			"149.154.167.91":      30 * time.Minute,    // Telegram
			"ping-chained-server": 1 * time.Nanosecond, // Internal ping-chained-server protocol
		}))
	} else {
		filterChain = filterChain.Append(tokenfilter.New(p.Token))
	}
	fd := common.NewRawFasttrackDomains(p.FasttrackDomains)
	rc, err := p.redisClientForPro()
	if err != nil {
		log.Debug("Not enabling pro because redis is not configured")
	}
	if rc != nil {
		filterChain = filterChain.Append(
			devicefilter.NewPre(redis.NewDeviceFetcher(rc), p.ThrottleThreshold, fd),
		)
	}
	filterChain = filterChain.Append(
		analytics.New(&analytics.Options{
			TrackingID:       p.ProxiedSitesTrackingID,
			SamplePercentage: p.ProxiedSitesSamplePercentage,
		}),
		devicefilter.NewPost(bl),
		commonfilter.New(&commonfilter.Options{
			AllowLocalhost: p.TestingLocal,
			Exceptions:     []string{"127.0.0.1:7300"},
		}),
		ping.New(0),
	)

	// Google anomaly detection can be triggered very often over IPv6.
	// Prefer IPv4 to mitigate, see issue #97
	dialer := preferIPV4Dialer(timeoutToDialOriginSite)
	dialerForPforward := dialer

	var rewriteConfigServerRequests func(*http.Request)
	if p.CfgSvrAuthToken != "" || p.CfgSvrDomains != "" {
		cfg := &configserverfilter.Options{
			AuthToken: p.CfgSvrAuthToken,
			Domains:   strings.Split(p.CfgSvrDomains, ","),
		}
		dialerForPforward = configserverfilter.Dialer(dialerForPforward, cfg)
		csf := configserverfilter.New(cfg)
		filterChain = filterChain.Append(csf)
		rewriteConfigServerRequests = csf.RewriteIfNecessary
	}

	pforwardOpts := &pforward.Options{
		IdleTimeout: p.IdleTimeout,
		Dialer:      dialerForPforward,
		OnRequest:   rewriteConfigServerRequests,
	}
	if bbrEnabled {
		pforwardOpts.OnResponse = bbr.AddMetrics
	}

	filterChain = filterChain.Append(
		// This filter will look for CONNECT requests and hijack those connections
		httpconnect.New(&httpconnect.Options{
			IdleTimeout:  p.IdleTimeout,
			AllowedPorts: p.allowedTunnelPorts(),
			Dialer:       dialer,
		}),
		// This filter will look for GET requests with X-Lantern-Persistent: true and
		// hijack those connections (new stateful HTTP connection management scheme).
		pforward.New(pforwardOpts),
		// This filter will handle all remaining HTTP requests (legacy HTTP
		// connection management scheme).
		forward.New(&forward.Options{
			IdleTimeout: p.IdleTimeout,
			Dialer:      dialer,
		}),
	)

	rc, err = p.redisClientForPro()
	if err != nil {
		log.Debug("Not enabling pro because redis is not configured")
		return filterChain, nil
	}

	if p.ServerID == "" {
		return nil, errors.New("Enabling Pro requires setting the \"serverid\" flag")
	}
	log.Debug("This proxy is configured to support Lantern Pro")
	proFilter, proErr := profilter.New(&profilter.Options{
		RedisClient:         rc,
		ServerID:            p.ServerID,
		KeepProTokenDomains: strings.Split(p.CfgSvrDomains, ","),
		FasttrackDomains:    fd,
	})
	if proErr != nil {
		return nil, errors.Wrap(proErr)
	}
	// Put profilter at the beginning of the chain.
	return filterChain.Prepend(proFilter), nil
}

func (p *Proxy) configureBandwidthReporting() *reportingConfig {
	rc, err := p.redisClientForReporting()
	if err != nil {
		log.Error(err)
	}
	var bordaReporter listeners.MeasuredReportFN
	if p.BordaReportInterval > 0 {
		bordaReporter = borda.Enable(p.BordaReportInterval, p.BordaSamplePercentage, p.BordaBufferSize)
	}
	return newReportingConfig(rc, p.EnableReports, bordaReporter)
}

func (p *Proxy) applyThrottling(srv *server.Server, rc *reportingConfig) error {
	if p.ThrottleBPS <= 0 && p.ThrottleThreshold <= 0 {
		log.Debug("Throttling is disabled")
		return nil
	}
	if p.ThrottleBPS <= 0 || p.ThrottleThreshold <= 0 {
		return errors.New("Throttling requires both throttlebps and throttlethreshold > 0")
	}
	if !rc.enabled {
		log.Debug("Not throttling because reporting is not enabled")
		return nil
	}

	log.Debugf("Throttling to %d bps after %d bytes", p.ThrottleBPS, p.ThrottleThreshold)
	// Add net.Listener wrappers for inbound connections
	srv.AddListenerWrappers(
		// Throttle connections when signaled
		func(ls net.Listener) net.Listener {
			return lanternlisteners.NewBitrateListener(ls, p.ThrottleBPS)
		},
	)
	return nil
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

func (p *Proxy) redisClientForPro() (redis.Client, error) {
	if p.RedisAddr == "" {
		return nil, errors.New("no redis address configured for pro")
	}
	redisOpts := &redis.Options{
		RedisURL:       p.RedisAddr,
		RedisCAFile:    p.RedisCA,
		ClientPKFile:   p.RedisClientPK,
		ClientCertFile: p.RedisClientCert,
	}
	return redis.GetClient(redisOpts)
}

func (p *Proxy) redisClientForReporting() (redis.Client, error) {
	if p.RedisAddr == "" {
		return nil, errors.New("no redis address configured for bandwidth reporting")
	}
	redisOpts := &redis.Options{
		RedisURL:       p.RedisAddr,
		RedisCAFile:    p.RedisCA,
		ClientPKFile:   p.RedisClientPK,
		ClientCertFile: p.RedisClientCert,
	}
	return redis.GetClient(redisOpts)
}

func (p *Proxy) serveOBFS4(srv *server.Server, bbrEnabled bool) {
	l, listenErr := p.listenTCP(p.Obfs4Addr, bbrEnabled)
	if listenErr != nil {
		log.Fatalf("Unable to listen for OBFS4 with tcp: %v", listenErr)
	}
	wrapped, wrapErr := obfs4listener.Wrap(l, p.Obfs4Dir)
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

func (p *Proxy) serveLampshade(srv *server.Server, bbrEnabled bool) {
	l, err := p.listenTCP(p.LampshadeAddr, bbrEnabled)
	if err != nil {
		log.Fatalf("Unable to listen for lampshade with tcp: %v", err)
	}
	wrapped, wrapErr := lampshade.Wrap(l, p.CertFile, p.KeyFile)
	if wrapErr != nil {
		log.Fatalf("Unable to initialize lampshade with tcp: %v", wrapErr)
	}
	go func() {
		serveErr := srv.Serve(wrapped, func(addr string) {
			log.Debugf("lampshade serving at %v", addr)
		})
		if serveErr != nil {
			log.Fatalf("Error serving lampshade at %v: %v", wrapped.Addr(), serveErr)
		}
	}()
}

func (p *Proxy) listenTCP(addr string, bbrEnabled bool) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if p.DiffServTOS > 0 {
		log.Debugf("Setting diffserv TOS to %d", p.DiffServTOS)
		// Note - this doesn't actually wrap the underlying connection, it'll still
		// be a net.TCPConn
		l = diffserv.Wrap(l, p.DiffServTOS)
	} else {
		log.Debugf("Not setting diffserv TOS")
	}
	if bbrEnabled {
		log.Debugf("Wrapping listener with BBR metrics support: %v", l.Addr())
		l = bbr.Wrap(l)
	}
	return l, nil
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
