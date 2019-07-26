package proxy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	rclient "gopkg.in/redis.v5"

	bordaClient "github.com/getlantern/borda/client"
	"github.com/getlantern/cmux"
	"github.com/getlantern/enhttp"
	"github.com/getlantern/errors"
	utp "github.com/getlantern/go-libutp"
	"github.com/getlantern/golog"
	"github.com/getlantern/gonat"
	"github.com/getlantern/kcpwrapper"
	"github.com/getlantern/ops"
	packetforward "github.com/getlantern/packetforward/server"
	"github.com/getlantern/pcapper"
	"github.com/getlantern/proxy"
	"github.com/getlantern/proxy/filters"
	"github.com/getlantern/quicwrapper"
	"github.com/getlantern/tinywss"
	"github.com/getlantern/tlsdefaults"
	"github.com/getlantern/tlsredis"

	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/proxyfilters"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/bbr"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/borda"
	"github.com/getlantern/http-proxy-lantern/cleanheadersfilter"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	"github.com/getlantern/http-proxy-lantern/diffserv"
	"github.com/getlantern/http-proxy-lantern/domains"
	"github.com/getlantern/http-proxy-lantern/googlefilter"
	"github.com/getlantern/http-proxy-lantern/httpsrewriter"
	"github.com/getlantern/http-proxy-lantern/instrument"
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
	"github.com/getlantern/http-proxy-lantern/wss"
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
	HTTPAddr                           string
	HTTPMultiplexAddr                  string
	HTTPUTPAddr                        string
	BordaReportInterval                time.Duration
	BordaSamplePercentage              float64
	BordaBufferSize                    int
	ExternalIP                         string
	CertFile                           string
	CfgSvrAuthToken                    string
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
	Obfs4MultiplexAddr                 string
	Obfs4UTPAddr                       string
	Obfs4Dir                           string
	Obfs4HandshakeConcurrency          int
	Obfs4MaxPendingHandshakesPerClient int
	Obfs4HandshakeTimeout              time.Duration
	KCPConf                            string
	Benchmark                          bool
	DiffServTOS                        int
	LampshadeAddr                      string
	LampshadeUTPAddr                   string
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
	WSSAddr                            string
	PCAPDir                            string
	PCAPIPs                            int
	PCAPSPerIP                         int
	PCAPSnapLen                        int
	PCAPTimeout                        time.Duration
	PacketForwardAddr                  string
	PacketForwardIntf                  string

	bm             bbr.Middleware
	rc             *rclient.Client
	throttleConfig throttle.Config
}

type listenerBuilderFN func(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error)

type addresses struct {
	obfs4          string
	obfs4Multiplex string
	http           string
	httpMultiplex  string
	lampshade      string
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	var onServerError func(conn net.Conn, err error)
	var onListenerError func(conn net.Conn, err error)
	if p.PCAPDir != "" && p.PCAPIPs > 0 && p.PCAPSPerIP > 0 {
		log.Debugf("Enabling packet capture, capturing the %d packets for each of the %d most recent IPs into %v", p.PCAPSPerIP, p.PCAPIPs, p.PCAPDir)
		pcapper.StartCapturing("http-proxy", "eth0", "/tmp", p.PCAPIPs, p.PCAPSPerIP, p.PCAPSnapLen, p.PCAPTimeout)
		onServerError = func(conn net.Conn, err error) {
			ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
			pcapper.Dump(ip, log.Errorf("Unexpected error handling traffic from %v: %v", ip, err).Error())
		}
		onListenerError = func(conn net.Conn, err error) {
			ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
			pcapper.Dump(ip, log.Errorf("Unexpected error handling new connection from %v: %v", ip, err).Error())
		}

		// Handle signals
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
			syscall.SIGUSR1)
		go func() {
			for {
				s := <-c
				if s == syscall.SIGUSR1 {
					pcapper.DumpAll("Full Dump")
				} else {
					log.Debug("Stopping server")
					os.Exit(0)
				}
			}
		}()
	}

	p.setupPacketForward()
	p.setupOpsContext()
	p.setBenchmarkMode()
	p.bm = bbr.New()
	if p.BBRUpstreamProbeURL != "" {
		go p.bm.ProbeUpstream(p.BBRUpstreamProbeURL)
	}
	p.initRedisClient()
	p.loadThrottleConfig()

	if p.ENHTTPAddr != "" {
		return p.ListenAndServeENHTTP()
	}

	// Only allow connections from remote IPs that are not blacklisted
	blacklist := p.createBlacklist()
	filterChain, dial, err := p.createFilterChain(blacklist)
	if err != nil {
		return err
	}

	if p.QUICAddr != "" {
		filterChain = filterChain.Prepend(quic.NewMiddleware())
	}
	if p.WSSAddr != "" {
		filterChain = filterChain.Append(wss.NewMiddleware())
	}
	filterChain = filterChain.Prepend(opsfilter.New(p.bm))

	bwReporting, bordaReporter := p.configureBandwidthReporting()

	srv := server.New(&server.Opts{
		IdleTimeout:              p.IdleTimeout,
		Dial:                     dial,
		Filter:                   instrument.WrapFilter("proxy", filterChain),
		OKDoesNotWaitForUpstream: !p.ConnectOKWaitsForUpstream,
		OnError:                  instrument.WrapConnErrorHandler("proxy_serve", onServerError),
	})
	// Although we include blacklist functionality, it's currently only used to
	// track potential blacklisting and doesn't actually blacklist anyone.
	srv.Allow = blacklist.OnConnect
	p.applyThrottling(srv, bwReporting)
	srv.AddListenerWrappers(bwReporting.wrapper)

	allListeners := make([]net.Listener, 0)
	addListenerIfNecessary := func(addr string, fn listenerBuilderFN) error {
		if addr == "" {
			return nil
		}
		l, err := fn(addr, bordaReporter)
		if err != nil {
			return err
		}
		allListeners = append(allListeners, l)
		return nil
	}

	addListenersForBaseTransport := func(baseListen func(string, bool) (net.Listener, error), addrs *addresses) error {
		if err := addListenerIfNecessary(addrs.obfs4, p.listenOBFS4(baseListen)); err != nil {
			return err
		}
		if err := addListenerIfNecessary(addrs.obfs4Multiplex, p.wrapMultiplexing(p.listenOBFS4(baseListen))); err != nil {
			return err
		}

		// We pass onListenerError to lampshade so that we can count errors in its
		// internal connection handling and dump pcaps in response to them.
		onListenerError = instrument.WrapConnErrorHandler("proxy_lampshade_listen", onListenerError)
		if err := addListenerIfNecessary(addrs.lampshade, p.listenLampshade(true, onListenerError, baseListen)); err != nil {
			return err
		}

		if err := addListenerIfNecessary(addrs.http, p.wrapTLSIfNecessary(p.listenHTTP(baseListen))); err != nil {
			return err
		}
		if err := addListenerIfNecessary(addrs.httpMultiplex, p.wrapMultiplexing(p.wrapTLSIfNecessary(p.listenHTTP(baseListen)))); err != nil {
			return err
		}
		return nil
	}

	if err := addListenerIfNecessary(p.KCPConf, p.wrapTLSIfNecessary(p.listenKCP)); err != nil {
		return err
	}
	if err := addListenerIfNecessary(p.QUICAddr, p.listenQUIC); err != nil {
		return err
	}
	if err := addListenerIfNecessary(p.WSSAddr, p.listenWSS); err != nil {
		return err
	}

	if err := addListenersForBaseTransport(p.listenTCP, &addresses{
		obfs4:          p.Obfs4Addr,
		obfs4Multiplex: p.Obfs4MultiplexAddr,
		lampshade:      p.LampshadeAddr,
		http:           p.HTTPAddr,
		httpMultiplex:  p.HTTPMultiplexAddr,
	}); err != nil {
		return err
	}

	if err := addListenersForBaseTransport(p.listenUTP, &addresses{
		obfs4:          "",
		obfs4Multiplex: p.Obfs4UTPAddr,
		lampshade:      p.LampshadeUTPAddr,
		http:           "",
		httpMultiplex:  p.HTTPUTPAddr,
	}); err != nil {
		return err
	}

	errCh := make(chan error, len(allListeners))
	for _, _l := range allListeners {
		l := _l
		go func() {
			log.Debugf("Serving at: %v", l.Addr())
			errCh <- srv.Serve(l, mimic.SetServerAddr)
		}()
	}

	return <-errCh
}

func (p *Proxy) ListenAndServeENHTTP() error {
	el, err := net.Listen("tcp", p.ENHTTPAddr)
	if err != nil {
		return errors.New("Unable to listen for encapsulated HTTP at %v: %v", p.ENHTTPAddr, err)
	}
	log.Debugf("Listening for encapsulated HTTP at %v", el.Addr())
	filterChain := filters.Join(tokenfilter.New(p.Token), instrument.WrapFilter("http_ping", ping.New(0)))
	enhttpHandler := enhttp.NewServerHandler(p.ENHTTPReapIdleTime, p.ENHTTPServerURL)
	server := &http.Server{
		Handler: filters.Intercept(enhttpHandler, instrument.WrapFilter("proxy", filterChain)),
	}
	return server.Serve(el)
}

func (p *Proxy) wrapTLSIfNecessary(fn listenerBuilderFN) listenerBuilderFN {
	return func(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
		l, err := fn(addr, bordaReporter)
		if err != nil {
			return nil, err
		}

		if p.HTTPS {
			l, err = tlslistener.Wrap(l, p.KeyFile, p.CertFile)
			if err != nil {
				return nil, err
			}

			log.Debugf("Using TLS on %v", l.Addr())
		}

		return l, nil
	}
}

func (p *Proxy) wrapMultiplexing(fn listenerBuilderFN) listenerBuilderFN {
	return func(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
		l, err := fn(addr, bordaReporter)
		if err != nil {
			return nil, err
		}

		l = cmux.Listen(&cmux.ListenOpts{
			Listener: l,
		})

		log.Debugf("Multiplexing on %v", l.Addr())
		return l, nil
	}
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
	if p.WSSAddr != "" {
		return "wss"
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

	if p.rc == nil {
		log.Debug("Not enabling bandwidth limiting")
	} else {
		filterChain = filterChain.Append(
			proxy.OnFirstOnly(devicefilter.NewPre(
				redis.NewDeviceFetcher(p.rc), p.throttleConfig, !p.Pro)),
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
		allowedLocalAddrs := []string{"127.0.0.1:7300"}
		if p.PacketForwardAddr != "" {
			allowedLocalAddrs = append(allowedLocalAddrs, p.PacketForwardAddr)
		}
		filterChain = filterChain.Append(proxyfilters.BlockLocal(allowedLocalAddrs))
	}
	filterChain = filterChain.Append(instrument.WrapFilter("http_ping", ping.New(0)))

	// Google anomaly detection can be triggered very often over IPv6.
	// Prefer IPv4 to mitigate, see issue #97
	_dialer := preferIPV4Dialer(timeoutToDialOriginSite)
	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		op := ops.Begin("dial_origin")
		defer op.End()

		start := time.Now()

		// resolve separately so that we can track the DNS resolution time
		resolveOp := ops.Begin("resolve_origin")
		resolvedAddr, resolveErr := net.ResolveTCPAddr(network, addr)
		if resolveErr != nil {
			resolveOp.FailIf(resolveErr)
			op.FailIf(resolveErr)
			resolveOp.End()
			return nil, resolveErr
		}
		op.Set("resolve_origin_time", bordaClient.Avg(time.Now().Sub(start).Seconds()))
		resolveOp.End()

		conn, dialErr := _dialer(ctx, network, resolvedAddr.String())
		if dialErr != nil {
			op.FailIf(dialErr)
			return nil, dialErr
		}
		op.Set("dial_origin_time", bordaClient.Avg(time.Now().Sub(start).Seconds()))

		return conn, nil
	}
	dialerForPforward := dialer

	var requestRewriters []func(*http.Request)
	rewriter := &httpsrewriter.Rewriter{}
	dialerForPforward = rewriter.Dialer(dialerForPforward)
	filterChain = filterChain.Append(rewriter)
	requestRewriters = append(requestRewriters, rewriter.RewriteIfNecessary)
	if p.CfgSvrAuthToken != "" {
		rewrite := requestModifier(func(req *http.Request) {
			if !domains.ConfigForRequest(req).AddConfigServerHeaders {
				return
			}
			req.Header.Set(common.CfgSvrAuthTokenHeader, p.CfgSvrAuthToken)
			ip, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
				return
			}
			req.Header.Set(common.CfgSvrClientIPHeader, ip)
			log.Debugf("Adding header to config-server request from %s to %s", ip, req.Host)
		})
		filterChain = filterChain.Append(rewrite)
		requestRewriters = append(requestRewriters, rewrite)
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
			if domains.ConfigForRequest(req).AddForwardedFor {
				// Only add X-Forwarded-For for certain domains
				return proxyfilters.AddForwardedFor(ctx, req, next)
			}
			return next(ctx, req)
		}),
		proxyfilters.RestrictConnectPorts(p.allowedTunnelPorts()),
		proxyfilters.RecordOp,
		cleanheadersfilter.New(), // IMPORTANT, this should be the last filter in the chain to avoid stripping any headers that other filters might need
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

func (p *Proxy) listenHTTP(baseListen func(string, bool) (net.Listener, error)) listenerBuilderFN {
	return func(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
		l, err := baseListen(addr, true)
		if err != nil {
			return nil, errors.New("Unable to listen for HTTP: %v", err)
		}
		log.Debugf("Listening for HTTP(S) at %v", l.Addr())
		return l, nil
	}
}

func (p *Proxy) listenOBFS4(baseListen func(string, bool) (net.Listener, error)) listenerBuilderFN {
	return func(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
		l, err := baseListen(addr, true)
		if err != nil {
			return nil, errors.New("Unable to listen for OBFS4: %v", err)
		}
		wrapped, err := obfs4listener.Wrap(l, p.Obfs4Dir, p.Obfs4HandshakeConcurrency, p.Obfs4MaxPendingHandshakesPerClient, p.Obfs4HandshakeTimeout)
		if err != nil {
			l.Close()
			return nil, errors.New("Unable to wrap listener with OBFS4: %v", err)
		}
		log.Debugf("Listening for OBFS4 at %v", wrapped.Addr())
		return wrapped, nil
	}
}

func (p *Proxy) listenLampshade(trackBBR bool, onListenerError func(net.Conn, error), baseListen func(string, bool) (net.Listener, error)) listenerBuilderFN {
	return func(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
		l, err := baseListen(addr, false)
		if err != nil {
			return nil, err
		}
		if bordaReporter != nil {
			log.Debug("Wrapping lampshade's TCP listener with measured reporting")
			l = listeners.NewMeasuredListener(l,
				measuredReportingInterval,
				borda.ConnectionTypedBordaReporter("physical", bordaReporter))
		}
		wrapped, wrapErr := lampshade.Wrap(l, p.CertFile, p.KeyFile, onListenerError)
		if wrapErr != nil {
			log.Fatalf("Unable to initialize lampshade with tcp: %v", wrapErr)
		}
		log.Debugf("Listening for lampshade at %v", wrapped.Addr())

		if trackBBR {
			// We wrap the lampshade listener itself so that we record BBR metrics on
			// close of virtual streams rather than the physical connection.
			wrapped = p.bm.Wrap(wrapped)
		}

		return wrapped, nil
	}
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

func (p *Proxy) listenUTP(addr string, wrapBBR bool) (net.Listener, error) {
	var l net.Listener
	var err error
	l, err = utp.NewSocket("udp", addr)
	if err != nil {
		return nil, err
	}

	if p.IdleTimeout > 0 {
		l = listeners.NewIdleConnListener(l, p.IdleTimeout)
	}

	return l, nil
}

func (p *Proxy) listenKCP(kcpConf string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
	cfg := &kcpwrapper.ListenerConfig{}
	file, err := os.Open(kcpConf) // For read access.
	if err != nil {
		return nil, errors.New("Unable to open KCPConf at %v: %v", p.KCPConf, err)
	}

	err = json.NewDecoder(file).Decode(cfg)
	file.Close()
	if err != nil {
		return nil, errors.New("Unable to decode KCPConf at %v: %v", p.KCPConf, err)
	}

	log.Debugf("Listening KCP at %v", cfg.Listen)
	return kcpwrapper.Listen(cfg, func(conn net.Conn) net.Conn {
		if p.IdleTimeout <= 0 {
			return conn
		}
		return listeners.WrapIdleConn(conn, p.IdleTimeout)
	})
}

func (p *Proxy) listenQUIC(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
	tlsConf, err := tlsdefaults.BuildListenerConfig(addr, p.KeyFile, p.CertFile)
	if err != nil {
		return nil, err
	}

	config := &quicwrapper.Config{
		MaxIncomingStreams: 1000,
	}

	l, err := quicwrapper.ListenAddr(p.QUICAddr, tlsConf, config)
	if err != nil {
		return nil, err
	}

	log.Debugf("Listening for quic at %v", l.Addr())
	return l, err
}

func (p *Proxy) listenWSS(addr string, bordaReporter listeners.MeasuredReportFN) (net.Listener, error) {
	l, err := p.listenTCP(addr, true)
	if err != nil {
		return nil, errors.New("Unable to listen for wss: %v", err)
	}

	if p.HTTPS {
		l, err = tlslistener.Wrap(l, p.KeyFile, p.CertFile)
		if err != nil {
			return nil, err
		}
		log.Debugf("Using TLS on %v", l.Addr())
	}
	opts := &tinywss.ListenOpts{
		Listener: l,
	}

	l, err = tinywss.ListenAddr(opts)
	if err != nil {
		return nil, err
	}

	log.Debugf("Listening for wss at %v", l.Addr())
	return l, err
}

func (p *Proxy) setupPacketForward() {
	if p.PacketForwardAddr == "" {
		return
	}
	l, err := net.Listen("tcp", p.PacketForwardAddr)
	if err != nil {
		log.Errorf("Unable to listen for packet forwarding at %v: %v", p.PacketForwardAddr, err)
		return
	}
	s, err := packetforward.NewServer(&packetforward.Opts{
		Opts: gonat.Opts{
			StatsInterval: 15 * time.Second,
			IFName:        p.PacketForwardIntf,
			IdleTimeout:   90 * time.Second,
			BufferDepth:   1000,
		},
		BufferPoolSize: 50 * 1024 * 1024,
	})
	if err != nil {
		log.Errorf("Error configuring packet forwarding: %v", err)
		return
	}
	log.Debugf("Listening for packet forwarding at %v", l.Addr())

	go func() {
		if err := s.Serve(l); err != nil {
			log.Errorf("Error serving packet forwarding: %v", err)
		}
	}()
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

type requestModifier func(req *http.Request)

func (f requestModifier) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	f(req)
	return next(ctx, req)
}
