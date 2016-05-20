package redis

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/golog"
	"github.com/getlantern/keyman"
)

var (
	log = golog.LoggerFor("redis")
	rcs = make(map[string]*redis.Client)
)

// Options provides options for configuring connectivity to Redis.
type Options struct {
	// RedisURL is the redis instance's URL in the form
	// redis://[user:pass@host:port]. Required.
	RedisURL string

	// RedisCAFile is a path to a PEM-encoded certificate for the CA that signs
	// the redis instance's server certificate. If not supplied, only the system
	// default trusted roots will be used.
	RedisCAFile string

	// ClientPKFile is a path to a PEM-encoded private key for the client to use
	// to authenticate itself to the redis stunnel. If not supplied, no client
	// authentication is performed.
	ClientPKFile string

	// ClientCertFile is a path to a PEM-encoded certificate for the client to use
	// to authenticate itself to the redis stunnel. If not supplied, no client
	// authentication is performed.
	ClientCertFile string

	// DialTimeout caps the amount of time we're willing to wait for a TCP
	// connection. Defaults to 30 seconds.
	DialTimeout time.Duration

	// TCPKeepAlive enables TCP keepalives on the connection to Redis.
	// If set to 0, keepalives are disabled.
	TCPKeepAlive time.Duration
}

func getClient(opts *Options) (*redis.Client, error) {
	u, err := url.Parse(opts.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse Redis address: %s", err)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("Please provide a Redis URL of the form 'redis://[user:pass@host:port]'")
	}

	if rc, ok := rcs[u.Host]; ok {
		return rc, nil
	}

	tlsConfig := &tls.Config{}

	if opts.RedisCAFile == "" {
		log.Debugf("Not using custom Redis CA")
	} else {
		log.Debugf("Adding custom Redis CA from: %v", opts.RedisCAFile)
		cert, err2 := keyman.LoadCertificateFromFile(opts.RedisCAFile)
		if err2 != nil {
			return nil, fmt.Errorf("Unable to load RedisCAFile: %v", err2)
		}
		tlsConfig.RootCAs = cert.PoolContainingCert()
	}

	if opts.ClientPKFile == "" || opts.ClientCertFile == "" {
		log.Debug("Not enabling client TLS authentication")
	} else {
		log.Debugf("Enabling client TLS authentication using pk %v and cert %v", opts.ClientPKFile, opts.ClientCertFile)
		cert, err2 := tls.LoadX509KeyPair(opts.ClientCertFile, opts.ClientPKFile)
		if err2 != nil {
			return nil, fmt.Errorf("Unable to load Client certificate/key pair: %v", err2)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	opt := redis.Options{
		Dialer: func() (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   opts.DialTimeout,
				KeepAlive: opts.TCPKeepAlive,
			}
			if dialer.Timeout == 0 {
				dialer.Timeout = 30 * time.Second
				log.Debugf("Defaulted dial timeout to %v", dialer.Timeout)
			}
			return tls.DialWithDialer(dialer, "tcp", u.Host, tlsConfig)
		},
	}
	if u.User != nil {
		redisPass, _ := u.User.Password()
		opt.Password = redisPass
	}

	rc := redis.NewClient(&opt)
	rcs[u.Host] = rc
	return rc, nil
}
