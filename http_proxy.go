package proxy

import (
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/ops"
	"github.com/getlantern/tlsredis"
	rclient "gopkg.in/redis.v5"

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
	"github.com/getlantern/http-proxy-lantern/googlefilter"
	"github.com/getlantern/http-proxy-lantern/lampshade"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/opsfilter"
	"github.com/getlantern/http-proxy-lantern/ping"
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
)

// Proxy is an HTTP proxy.
type Proxy struct {
	TestingLocal                   bool
	Addr                           string
	BordaReportInterval            time.Duration
	BordaSamplePercentage          float64
	BordaBufferSize                int
	ExternalIP                     string
	CertFile                       string
	CfgSvrAuthToken                string
	CfgSvrDomains                  string
	EnableReports                  bool
	HTTPS                          bool
	IdleTimeout                    time.Duration
	KeyFile                        string
	Pro                            bool
	ProxiedSitesSamplePercentage   float64
	ProxiedSitesTrackingID         string
	ReportingRedisAddr             string
	ReportingRedisCA               string
	ReportingRedisClientPK         string
	ReportingRedisClientCert       string
	ThrottleRefreshInterval        time.Duration
	Token                          string
	TunnelPorts                    string
	Obfs4Addr                      string
	Obfs4Dir                       string
	Benchmark                      bool
	FasttrackDomains               string
	DiffServTOS                    int
	LampshadeAddr                  string
	VersionCheck                   bool
	VersionCheckMinVersion         string
	VersionCheckRedirectURL        string
	VersionCheckRedirectPercentage float64
	GoogleSearchRegex              string
	GoogleCaptchaRegex             string

	bm             bbr.Middleware
	rc             *rclient.Client
	throttleConfig throttle.Config
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	p.setupOpsContext()
	p.setBenchmarkMode()
	p.bm = bbr.New()
	p.initRedisClient()
	p.loadThrottleConfig()

	// Only allow connections from remote IPs that are not blacklisted
	blacklist := p.createBlacklist()
	filterChain, err := p.createFilterChain(blacklist)
	if err != nil {
		return err
	}

	bwReporting := p.configureBandwidthReporting()
	srv := server.NewServer(filterChain.Prepend(opsfilter.New(p.bm)))
	srv.Allow = blacklist.OnConnect
	p.applyThrottling(srv, bwReporting)
	srv.AddListenerWrappers(bwReporting.wrapper)
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, p.IdleTimeout)
		},
	)

	if p.Obfs4Addr != "" {
		p.serveOBFS4(srv)
	}

	l, err := p.listenTCP(p.Addr, true)
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
			p.serveLampshade(srv)
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

func (p *Proxy) createFilterChain(bl *blacklist.Blacklist) (filters.Chain, error) {
	filterChain := filters.Join(p.bm)

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

	if p.rc == nil {
		log.Debug("Not enabling bandwidth limiting")
	} else {
		fd := common.NewRawFasttrackDomains(p.FasttrackDomains)
		filterChain = filterChain.Append(
			devicefilter.NewPre(redis.NewDeviceFetcher(p.rc), p.throttleConfig, fd),
		)
	}

	filterChain = filterChain.Append(
		googlefilter.New(p.GoogleSearchRegex, p.GoogleCaptchaRegex),
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

	// check if the client is running below a certain version, and if true,
	// rewrite certain percentage of the requests to an URL to notify user.
	if p.VersionCheck {
		log.Debugf("versioncheck: Will rewrite %.4f%% of browser requests from clients below %s to %s",
			p.VersionCheckRedirectPercentage*100,
			p.VersionCheckMinVersion,
			p.VersionCheckRedirectURL,
		)
		vc := versioncheck.New(p.VersionCheckMinVersion,
			p.VersionCheckRedirectURL,
			[]string{"80"}, // checks CONNECT tunnel to 80 port only.
			p.VersionCheckRedirectPercentage)
		requestRewriters = append(requestRewriters, vc.RewriteIfNecessary)
		dialerForPforward = vc.Dialer(dialerForPforward)
		filterChain = filterChain.Append(vc.Filter())
	}

	var rewrite func(req *http.Request)
	if len(requestRewriters) > 0 {
		rewrite = func(req *http.Request) {
			for _, rw := range requestRewriters {
				rw(req)
			}
		}
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
		pforward.New(&pforward.Options{
			IdleTimeout: p.IdleTimeout,
			Dialer:      dialerForPforward,
			OnRequest:   rewrite,
			OnResponse:  p.bm.AddMetrics,
		}),
		// This filter will handle all remaining HTTP requests (legacy HTTP
		// connection management scheme).
		forward.New(&forward.Options{
			IdleTimeout: p.IdleTimeout,
			Dialer:      dialer,
		}),
	)

	return filterChain, nil
}

func (p *Proxy) configureBandwidthReporting() *reportingConfig {
	var bordaReporter listeners.MeasuredReportFN
	if p.BordaReportInterval > 0 {
		bordaReporter = borda.Enable(p.BordaReportInterval, p.BordaSamplePercentage, p.BordaBufferSize)
	}
	return newReportingConfig(p.rc, p.EnableReports, bordaReporter)
}

func (p *Proxy) loadThrottleConfig() {
	if p.Pro || p.rc == nil {
		log.Debug("Not loading throttle config")
		return
	}

	var err error
	p.throttleConfig, err = throttle.NewRedisConfig(p.rc, p.ThrottleRefreshInterval)
	if err != nil {
		p.throttleConfig = nil
		log.Errorf("Unable to read throttling config from redis, will not throttle: %v", err)
	}
}

func (p *Proxy) applyThrottling(srv *server.Server, rc *reportingConfig) {
	if p.Pro || p.throttleConfig == nil {
		log.Debug("Throttling is disabled")
	}
	if !rc.enabled {
		log.Debug("Not throttling because reporting is not enabled")
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

func (p *Proxy) serveLampshade(srv *server.Server) {
	l, err := p.listenTCP(p.LampshadeAddr, false)
	if err != nil {
		log.Fatalf("Unable to listen for lampshade with tcp: %v", err)
	}
	wrapped, wrapErr := lampshade.Wrap(l, p.CertFile, p.KeyFile)
	if wrapErr != nil {
		log.Fatalf("Unable to initialize lampshade with tcp: %v", wrapErr)
	}
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
