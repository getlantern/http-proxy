package proxy

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"time"

	_redis "gopkg.in/redis.v3"

	"github.com/getlantern/golog"
	"github.com/getlantern/measured"
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
	"github.com/getlantern/http-proxy-lantern/kcplistener"
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
	timeoutToDialOriginSite   = 10 * time.Second
	measuredReportingInterval = 1 * time.Minute
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
	EnablePro                    bool
	EnableReports                bool
	HTTPS                        bool
	IdleClose                    uint64
	KeyFile                      string
	MaxConns                     uint64
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
	Obfs4KCPAddr                 string
	Obfs4Dir                     string
	Benchmark                    bool
	FasttrackDomains             string
	DiffServTOS                  int
	LampshadeAddr                string
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	var err error
	ops.SetGlobal("app", "http-proxy")
	if p.ExternalIP != "" {
		log.Debugf("Will report with proxy_host: %v", p.ExternalIP)
		ops.SetGlobal("proxy_host", p.ExternalIP)
	}

	if p.Benchmark {
		log.Debug("Putting proxy into benchmarking mode. Only a limited rate of requests to a specific set of domains will be allowed, no authentication token required.")
		p.HTTPS = true
		p.Token = "bench"
	}

	reportMeasured := func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		// do nothing by default
	}

	// Get a Redis client
	var rc *_redis.Client
	if p.RedisAddr != "" {
		redisOpts := &redis.Options{
			RedisURL:       p.RedisAddr,
			RedisCAFile:    p.RedisCA,
			ClientPKFile:   p.RedisClientPK,
			ClientCertFile: p.RedisClientCert,
		}
		var redisErr error
		rc, redisErr = redis.GetClient(redisOpts)
		if redisErr != nil {
			log.Fatal(redisErr)
		}
	}

	shouldReport := p.EnableReports && rc != nil

	// Reporting
	if shouldReport {
		reportMeasured = redis.NewMeasuredReporter(rc, measuredReportingInterval)
	}

	// Throttling
	if p.ThrottleBPS > 0 && p.ThrottleThreshold > 0 {
		if !shouldReport {
			log.Debug("Not throttling because reporting is not enabled")
		} else {
			log.Debugf("Throttling to %d bps after %d bytes", p.ThrottleBPS, p.ThrottleThreshold)
		}
	} else if (p.ThrottleBPS > 0) != (p.ThrottleThreshold > 0) {
		log.Fatal("Throttling requires both throttlebps and throttlethreshold > 0")
	} else {
		log.Debug("Throttling is disabled")
	}

	// Configure borda
	if p.BordaReportInterval > 0 {
		oldReportMeasured := reportMeasured
		bordaReportMeasured := borda.Enable(p.BordaReportInterval, p.BordaSamplePercentage, p.BordaBufferSize)
		reportMeasured = func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
			oldReportMeasured(ctx, stats, deltaStats, final)
			bordaReportMeasured(ctx, stats, deltaStats, final)
		}
	}

	// Set up a blacklist
	bl := blacklist.New(blacklist.Options{
		MaxIdleTime:        30 * time.Second,
		MaxConnectInterval: 5 * time.Second,
		AllowedFailures:    10,
		Expiration:         6 * time.Hour,
	})

	idleTimeout := time.Duration(p.IdleClose) * time.Second
	var allowedPorts []int
	if p.TunnelPorts != "" {
		allowedPorts, err = portsFromCSV(p.TunnelPorts)
		if err != nil {
			log.Fatal(err)
		}
	}

	var filterChain filters.Chain
	var bbrfilter bbr.Filter
	var bbrOnResponse func(*http.Response) *http.Response

	if runtime.GOOS == "linux" {
		log.Debug("Tracking bbr metrics")
		bbrfilter = bbr.New()
		bbrOnResponse = bbrfilter.OnResponse
		filterChain = filterChain.Append(bbrfilter)
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

	var attachConfigServerHeader func(*http.Request)
	if p.CfgSvrAuthToken != "" || p.CfgSvrDomains != "" {
		csf := configserverfilter.New(&configserverfilter.Options{
			AuthToken: p.CfgSvrAuthToken,
			Domains:   strings.Split(p.CfgSvrDomains, ","),
		})
		filterChain = filterChain.Append(csf)
		attachConfigServerHeader = csf.AttachHeaderIfNecessary
	}

	// Google anomaly detection can be triggered very often over IPv6.
	// Prefer IPv4 to mitigate, see issue #97
	dialer := preferIPV4Dialer(timeoutToDialOriginSite)
	filterChain = filterChain.Append(
		// This filter will look for CONNECT requests and hijack those connections
		httpconnect.New(&httpconnect.Options{
			IdleTimeout:  idleTimeout,
			AllowedPorts: allowedPorts,
			Dialer:       dialer,
		}),
		// This filter will look for GET requests with X-Lantern-Persistent: true and
		// hijack those connections (new stateful HTTP connection management scheme).
		pforward.New(&pforward.Options{
			IdleTimeout: idleTimeout,
			Dialer:      dialer,
			OnRequest:   attachConfigServerHeader,
			OnResponse:  bbrOnResponse,
		}),
		// This filter will handle all remaining HTTP requests (legacy HTTP
		// connection management scheme).
		forward.New(&forward.Options{
			IdleTimeout: idleTimeout,
			Dialer:      dialer,
		}),
	)

	// Pro support
	if p.EnablePro {
		if rc == nil {
			log.Debug("Not enabling pro because redis is not configured")
		} else {
			if p.ServerID == "" {
				log.Fatal("Enabling Pro requires setting the \"serverid\" flag")
			}
			log.Debug("This proxy is configured to support Lantern Pro")
			proFilter, proErr := profilter.New(&profilter.Options{
				RedisClient:         rc,
				ServerID:            p.ServerID,
				KeepProTokenDomains: strings.Split(p.CfgSvrDomains, ","),
				FasttrackDomains:    fd,
			})
			if proErr != nil {
				log.Fatal(proErr)
			}

			// Put profilter at the beginning of the chain.
			filterChain = filterChain.Prepend(proFilter)
		}
	}

	srv := server.NewServer(filterChain.Prepend(opsfilter.New()))
	// Only allow connections from remote IPs that are not blacklisted
	srv.Allow = bl.OnConnect

	// Add net.Listener wrappers for inbound connections
	if p.ThrottleBPS > 0 {
		srv.AddListenerWrappers(
			// Throttle connections when signaled
			func(ls net.Listener) net.Listener {
				return lanternlisteners.NewBitrateListener(ls, p.ThrottleBPS)
			},
		)
	}
	if shouldReport || p.BordaReportInterval > 0 {
		srv.AddListenerWrappers(
			// Measure connections
			func(ls net.Listener) net.Listener {
				return listeners.NewMeasuredListener(ls, measuredReportingInterval, reportMeasured)
			},
		)
	}
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, idleTimeout)
		},
	)

	onAddress := func(addr string) {
		mimic.SetServerAddr(addr)
	}

	serveOBFS4 := func(wrapped net.Listener) {
		l, wrapErr := obfs4listener.Wrap(wrapped, p.Obfs4Dir)
		if wrapErr != nil {
			log.Fatalf("Unable to listen with obfs4 at %v: %v", wrapped.Addr(), wrapErr)
		}
		log.Debugf("Listening for OBFS4 at %v", l.Addr())

		go func() {
			serveErr := srv.Serve(l, func(addr string) {
				log.Debugf("obfs4 serving at %v", addr)
			})
			if serveErr != nil {
				log.Fatalf("Error serving obfs4 at %v: %v", wrapped.Addr(), serveErr)
			}
		}()
	}

	if p.Obfs4Addr != "" {
		l, listenErr := p.listenTCP(p.Obfs4Addr, bbrfilter)
		if listenErr != nil {
			log.Fatalf("Unable to listen for OBFS4 with tcp: %v", listenErr)
		}
		serveOBFS4(l)
	}

	if p.Obfs4KCPAddr != "" {
		l, listenErr := kcplistener.NewListener(p.Obfs4KCPAddr)
		if listenErr != nil {
			log.Fatalf("Unable to listen for OBFS4 with kcp: %v", listenErr)
		}
		serveOBFS4(l)
	}

	l, err := p.listenTCP(p.Addr, bbrfilter)
	if err != nil {
		return fmt.Errorf("Unable to listen HTTP: %v", err)
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
			ll, listenErr := p.listenTCP(p.LampshadeAddr, bbrfilter)
			if listenErr != nil {
				log.Fatalf("Unable to listen for lampshade with tcp: %v", listenErr)
			}
			ll, listenErr = lampshade.Wrap(ll, p.CertFile, p.KeyFile)
			if listenErr != nil {
				log.Fatalf("Unable to initialize lampshade with tcp: %v", listenErr)
			}
			go srv.Serve(ll, func(addr string) {
				log.Debugf("lampshade serving at %v", addr)
			})
		}
	}

	log.Debugf("Listening for %v at %v", protocol, l.Addr())
	err = srv.Serve(l, onAddress)
	if err != nil {
		log.Errorf("Error serving HTTP(S): %v", err)
	}
	return err
}

func (p *Proxy) listenTCP(addr string, bbrfilter bbr.Filter) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if p.DiffServTOS > 0 {
		log.Debugf("Setting diffserv TOS to %d", p.DiffServTOS)
		// Note - this doesn't actually wrap the underlying connection, it'll still
		// be a net.TCPConn
		l = diffserv.Wrap(l, p.DiffServTOS)
	}
	if bbrfilter != nil {
		log.Debugf("Wrapping listener with BBR metrics support: %v", l.Addr())
		l = bbrfilter.Wrap(l)
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
