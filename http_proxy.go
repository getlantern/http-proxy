package proxy

import (
	"net"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/measured"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy/commonfilter"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/http-proxy/forward"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/borda"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/opsfilter"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/profilter"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
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
}

// ListenAndServe listens, serves and blocks.
func (p *Proxy) ListenAndServe() error {
	var err error
	ops.SetGlobal("app", "http-proxy")

	var m *measured.Measured
	redisOpts := &redis.Options{
		RedisURL:       p.RedisAddr,
		RedisCAFile:    p.RedisCA,
		ClientPKFile:   p.RedisClientPK,
		ClientCertFile: p.RedisClientCert,
	}

	// Reporting
	if p.EnableReports {
		rp, reporterErr := redis.NewMeasuredReporter(redisOpts)
		if reporterErr != nil {
			log.Fatalf("Error creating measured reporter: %v", reporterErr)
		}
		m = measured.New(5000)
		m.Start(time.Minute, rp)
		defer m.Stop()
	}

	// Throttling
	if (p.ThrottleBPS > 0 || p.ThrottleThreshold > 0) &&
		(p.ThrottleBPS <= 0 || p.ThrottleThreshold <= 0) &&
		!p.EnableReports {
		log.Fatal("Throttling requires reports enabled and both throttlebps and throttlethreshold > 0")
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
		allowedPorts, err = httpconnect.AllowedPortsFromCSV(p.TunnelPorts)
		if err != nil {
			log.Fatal(err)
		}
	}

	filterChain := filters.Join(
		tokenfilter.New(p.Token),
		devicefilter.NewPre(p.ThrottleThreshold),
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

	filterChain = filterChain.Append(
		httpconnect.New(&httpconnect.Options{
			IdleTimeout:  idleTimeout,
			AllowedPorts: allowedPorts,
		}),
		forward.New(&forward.Options{
			IdleTimeout: idleTimeout,
		}),
	)

	// Pro support
	if p.EnablePro {
		if p.ServerID == "" {
			log.Fatal("Enabling Pro requires setting the \"serverid\" flag")
		}
		log.Debug("This proxy is configured to support Lantern Pro")
		proFilter, err := profilter.New(&profilter.Options{
			RedisOpts: redisOpts,
			ServerID:  p.ServerID,
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
				return listeners.NewMeasuredListener(ls, 100*time.Millisecond, m)
			},
		)
	}
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, idleTimeout)
		},
		// Preprocess connection to issue custom errors before they are passed to the server
		func(ls net.Listener) net.Listener {
			return lanternlisteners.NewPreprocessorListener(ls)
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
				log.Fatalf("Error serving OBFS4: %v", err)
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
