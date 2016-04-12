package httpproxylantern

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy/commonfilter"
	"github.com/getlantern/http-proxy/forward"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/obfs4listener"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/profilter"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
)

var (
	log = golog.LoggerFor("lantern-proxy")
)

type Server struct {
	TestingLocal                 bool
	Addr                         string
	CertFile                     string
	CfgSvrAuthToken              string
	CfgSvrDomains                string
	EnablePro                    bool
	EnableReports                bool
	HTTPS                        bool
	IdleClose                    uint64
	Keyfile                      string
	MaxConns                     uint64
	ProxiedSitesSamplePercentage float64
	ProxiedSitesTrackingId       string
	RedisAddr                    string
	ServerId                     string
	Token                        string
	TunnelPorts                  string
	Obfs4Addr                    string
	Obfs4Dir                     string
}

func (s *Server) ListenAndServe() error {
	var err error

	// Reporting
	if s.EnableReports {
		rp, err := redis.NewMeasuredReporter(s.RedisAddr)
		if err != nil {
			log.Errorf("Error connecting to redis: %v", err)
		} else {
			measured.Start(20*time.Second, rp)
			defer measured.Stop()
		}
	}

	// Middleware
	forwarder, err := forward.New(nil, forward.IdleTimeoutSetter(time.Duration(s.IdleClose)*time.Second))
	if err != nil {
		return err
	}

	var nextFilter http.Handler = forwarder

	if s.TunnelPorts != "" {
		nextFilter, err = httpconnect.New(forwarder,
			httpconnect.IdleTimeoutSetter(time.Duration(s.IdleClose)*time.Second),
			httpconnect.AllowedPortsFromCSV(s.TunnelPorts))
	} else {
		nextFilter, err = httpconnect.New(forwarder,
			httpconnect.IdleTimeoutSetter(time.Duration(s.IdleClose)*time.Second))
	}
	if err != nil {
		return err
	}

	if s.CfgSvrAuthToken != "" || s.CfgSvrDomains != "" {
		domains := strings.Split(s.CfgSvrDomains, ",")
		nextFilter, err = configserverfilter.New(nextFilter,
			configserverfilter.AuthToken(s.CfgSvrAuthToken),
			configserverfilter.Domains(domains))
		if err != nil {
			return err
		}
	}

	pingFilter := ping.New(nextFilter)

	commonFilter, err := commonfilter.New(pingFilter,
		s.TestingLocal,
		commonfilter.SetException("127.0.0.1:7300"),
	)
	if err != nil {
		return err
	}

	deviceFilterPost := devicefilter.NewPost(commonFilter)

	analyticsFilter := analytics.New(s.ProxiedSitesTrackingId, s.ProxiedSitesSamplePercentage, deviceFilterPost)

	deviceFilterPre, err := devicefilter.NewPre(analyticsFilter)
	if err != nil {
		log.Fatal(err)
	}

	tokenFilter, err := tokenfilter.New(deviceFilterPre, tokenfilter.TokenSetter(s.Token))
	if err != nil {
		return err
	}

	var srv *server.Server

	// Pro support
	if s.EnablePro {
		if s.ServerId == "" {
			return fmt.Errorf("Enabling Pro requires setting the \"serverid\" flag")
		}
		log.Debug("This proxy is configured to support Lantern Pro")
		proFilter, err := profilter.New(tokenFilter,
			profilter.RedisConfigSetter(s.RedisAddr, s.ServerId),
		)
		if err != nil {
			return err
		}

		srv = server.NewServer(proFilter)
	} else {
		srv = server.NewServer(tokenFilter)
	}

	// Add net.Listener wrappers for inbound connections
	if s.EnableReports {
		srv.AddListenerWrappers(
			// Measure connections
			func(ls net.Listener) net.Listener {
				return listeners.NewMeasuredListener(ls, 100*time.Millisecond)
			},
		)
	}
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, time.Duration(s.IdleClose)*time.Second)
		},
		// Preprocess connection to issue custom errors before they are passed to the server
		func(ls net.Listener) net.Listener {
			return lanternlisteners.NewPreprocessorListener(ls)
		},
	)

	initMimic := func(addr string) {
		mimic.SetServerAddr(addr)
	}

	errCh := make(chan error)

	if s.Obfs4Addr != "" {
		l, err := obfs4listener.NewListener(s.Obfs4Addr, s.Obfs4Dir)
		if err != nil {
			return fmt.Errorf("Unable to listen with obfs4: %v", err)
		}
		go func() {
			err := srv.Serve(l, func(addr string) {
				log.Debugf("obfs4 listening at %v", addr)
			})
			if err != nil {
				errCh <- fmt.Errorf("Error serving OBFS4: %v", err)
			}
		}()
	}
	go func() {
		if s.HTTPS {
			err = srv.ListenAndServeHTTPS(s.Addr, s.Keyfile, s.CertFile, initMimic)
		} else {
			err = srv.ListenAndServeHTTP(s.Addr, initMimic)
		}
		if err != nil {
			errCh <- fmt.Errorf("Error serving HTTP(S): %v", err)
		}
	}()

	return <-errCh
}
