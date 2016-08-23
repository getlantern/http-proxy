package proxy

import (
	"net"
	_ "net/http/pprof"
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
	"github.com/getlantern/http-proxy/ratelimiter"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/borda"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	"github.com/getlantern/http-proxy-lantern/kcplistener"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/opsfilter"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/profilter"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
)

const timeoutToDialOriginSite = 10 * time.Second

var (
	log = golog.LoggerFor("lantern-proxy")
)

// Proxy is an HTTP proxy.
type Proxy struct {
	TestingLocal                 bool
	Addr                         string
	BordaReportInterval          time.Duration
	BordaSamplePercentage        float64
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
	Obfs4Dir                     string
	KCPAddr                      string
	Benchmark                    bool
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	var err error
	ops.SetGlobal("app", "http-proxy")

	if p.Benchmark {
		log.Debug("Putting proxy into benchmarking mode. Only a limited rate of requests to a specific set of domains will be allowed, no authentication token required.")
		p.HTTPS = true
		p.Token = "bench"
		p.KCPAddr = ":10000"
	}

	var m *measured.Measured
	// Get a Redis client
	var rc *_redis.Client
	if p.RedisAddr != "" {
		redisOpts := &redis.Options{
			RedisURL:       p.RedisAddr,
			RedisCAFile:    p.RedisCA,
			ClientPKFile:   p.RedisClientPK,
			ClientCertFile: p.RedisClientCert,
		}
		var err error
		rc, err = redis.GetClient(redisOpts)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Reporting
	if p.EnableReports {
		rp := redis.NewMeasuredReporter(rc)
		m = measured.New(5000)
		m.Start(time.Minute, rp)
		defer m.Stop()
	}

	// Throttling
	if p.ThrottleBPS > 0 && p.ThrottleThreshold > 0 {
		if !p.EnableReports {
			log.Fatal("Throttling requires reporting enabled")
		}
		log.Debugf("Throttling to %d bps after %d bytes", p.ThrottleBPS, p.ThrottleThreshold)
	} else if (p.ThrottleBPS > 0) != (p.ThrottleThreshold > 0) {
		log.Fatal("Throttling requires both throttlebps and throttlethreshold > 0")
	} else {
		log.Debug("Throttling is disabled")
	}

	// Configure borda
	if p.BordaReportInterval > 0 {
		borda.Enable(p.BordaReportInterval, p.BordaSamplePercentage)
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
	if p.Benchmark {
		filterChain = filterChain.Append(ratelimiter.New(5000, map[string]time.Duration{
			"www.google.com":      30 * time.Minute,
			"www.facebook.com":    30 * time.Minute,
			"67.media.tumblr.com": 30 * time.Minute,
			"i.ytimg.com":         30 * time.Minute, // YouTube play button
			"149.154.167.91":      30 * time.Minute, // Telegram
		}))
	} else {
		filterChain = filters.Join(tokenfilter.New(p.Token))
	}
	if rc != nil {
		filterChain = filterChain.Append(
			devicefilter.NewPre(redis.NewDeviceFetcher(rc), p.ThrottleThreshold),
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
		ping.New(),
	)

	if p.CfgSvrAuthToken != "" || p.CfgSvrDomains != "" {
		filterChain = filterChain.Append(configserverfilter.New(&configserverfilter.Options{
			AuthToken: p.CfgSvrAuthToken,
			Domains:   strings.Split(p.CfgSvrDomains, ","),
		}))
	}

	// Google anomaly detection can be triggered very often over IPv6.
	// Prefer IPv4 to mitigate, see issue #97
	dialer := preferIPV4Dialer(timeoutToDialOriginSite)
	filterChain = filterChain.Append(
		httpconnect.New(&httpconnect.Options{
			IdleTimeout:  idleTimeout,
			AllowedPorts: allowedPorts,
			Dialer:       dialer,
		}),
		forward.New(&forward.Options{
			IdleTimeout: idleTimeout,
			Dialer:      dialer,
		}),
	)

	// Pro support
	if p.EnablePro {
		if p.ServerID == "" {
			log.Fatal("Enabling Pro requires setting the \"serverid\" flag")
		}
		log.Debug("This proxy is configured to support Lantern Pro")
		proFilter, err := profilter.New(&profilter.Options{
			RedisClient:         rc,
			ServerID:            p.ServerID,
			KeepProTokenDomains: strings.Split(p.CfgSvrDomains, ","),
		})
		if err != nil {
			log.Fatal(err)
		}

		// Put profilter at the beginning of the chain.
		filterChain = filterChain.Prepend(proFilter)
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
	if p.EnableReports {
		srv.AddListenerWrappers(
			// Measure connections
			func(ls net.Listener) net.Listener {
				return listeners.NewMeasuredListener(ls, 10*time.Second, m)
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
		proxyHost, proxyPort, err2 := net.SplitHostPort(addr)
		if err2 == nil {
			ops.SetGlobal("proxy_host", proxyHost)
			ops.SetGlobal("proxy_port", proxyPort)
		}
		mimic.SetServerAddr(addr)
	}

	if p.Obfs4Addr != "" {
		l, err := obfs4listener.NewListener(p.Obfs4Addr, p.Obfs4Dir)
		if err != nil {
			log.Fatalf("Unable to listen with obfs4: %v", err)
		}
		go func() {
			err := srv.Serve(l, func(addr string) {
				log.Debugf("obfs4 listening at %v", addr)
			})
			if err != nil {
				log.Fatalf("Error serving obfs4: %v", err)
			}
		}()
	}
	if p.KCPAddr != "" {
		l, err := kcplistener.NewListener(p.KCPAddr, p.KeyFile, p.CertFile)
		if err != nil {
			log.Fatalf("Unable to listen with kcp: %v", err)
		}
		go func() {
			err := srv.Serve(l, func(addr string) {
				log.Debugf("kcp listening at %v", addr)
			})
			if err != nil {
				log.Fatalf("Error serving kcp: %v", err)
			}
		}()
	}
	if p.HTTPS {
		err = srv.ListenAndServeHTTPS(p.Addr, p.KeyFile, p.CertFile, onAddress)
	} else {
		err = srv.ListenAndServeHTTP(p.Addr, onAddress)
	}
	if err != nil {
		log.Errorf("Error serving HTTP(S): %v", err)
	}
	return err
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
