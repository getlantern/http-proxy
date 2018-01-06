// Add required headers to config-server requests and change the scheme to HTTPS.
// Ref https://github.com/getlantern/config-server/issues/4

package configserverfilter

import (
	"errors"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("configServerFilter")

type Options struct {
	AuthToken string
	Domains   []string
}

type ConfigServerFilter struct {
	opts          *Options
	dnsCache      map[string]string
	dnsCacheMutex sync.RWMutex

	ips      map[string]bool
	ipsMutex sync.RWMutex

	consecFailures int32
	refreshingDNS  bool
	random         *rand.Rand
}

// New creaetes a new filter for config server requests.
func New(opts *Options) *ConfigServerFilter {
	if opts.AuthToken == "" || len(opts.Domains) == 0 {
		panic(errors.New("should set both config-server auth token and domains"))
	}
	log.Debugf("Will attach %s header on GET requests to %+v", common.CfgSvrAuthTokenHeader, opts.Domains)

	csf := &ConfigServerFilter{
		opts:     opts,
		dnsCache: make(map[string]string),
		ips:      make(map[string]bool),
		random:   rand.New(rand.NewSource(time.Now().Unix())),
	}
	csf.refreshDNSCache()
	go csf.clearIPs()
	return csf
}

func (f *ConfigServerFilter) clearIPs() {
	for {
		time.Sleep(1 * time.Hour)
		f.ipsMutex.Lock()
		f.ips = make(map[string]bool)
		f.ipsMutex.Unlock()
	}
}

func (f *ConfigServerFilter) refreshDNSCache() {
	f.dnsCacheMutex.Lock()
	defer func() {
		atomic.StoreInt32(&f.consecFailures, 0)
		f.refreshingDNS = false
		f.dnsCacheMutex.Unlock()
	}()
	if f.refreshingDNS {
		return
	}
	f.refreshingDNS = true
	for _, domain := range f.opts.Domains {
		f.dnsCache[domain] = f.resolveDomain(domain)
	}
}

func (f *ConfigServerFilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	ip, cached := f.notModified(req)
	if cached {
		return &http.Response{StatusCode: http.StatusNotModified}, nil, nil
	}
	f.RewriteIfNecessary(req)

	resp, nextCtx, err := next(ctx, req)
	if err != nil && resp == nil {
		log.Errorf("Error hitting config server...refreshing DNS cache %v", err)
		f.handleFailure()
		return resp, nextCtx, err
	}

	if resp != nil && resp.StatusCode >= 500 && resp.StatusCode < 600 {
		f.handleFailure()
	} else if ip != "" && resp.StatusCode == http.StatusOK {
		f.ipsMutex.Lock()
		f.ips[ip] = true
		f.ipsMutex.Unlock()
	}

	return resp, nextCtx, err
}

func (f *ConfigServerFilter) notModified(req *http.Request) (string, bool) {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return "", false
	}
	f.ipsMutex.RLock()
	_, ok := f.ips[ip]
	f.ipsMutex.RUnlock()
	return ip, ok
}

func (f *ConfigServerFilter) handleFailure() {
	// If we have enough consecutive failures, try another config server.
	cf := atomic.AddInt32(&f.consecFailures, 1)
	if cf > 10 {
		log.Debugf("Too many consecutive failures...refreshing DNS cache")
		f.refreshDNSCache()
	}
}

func (f *ConfigServerFilter) RewriteIfNecessary(req *http.Request) {
	// It's unlikely that config-server will add non-GET public endpoint.
	// Bypass all other methods, especially CONNECT (https).
	if req.Method == "GET" {
		if matched := in(req.Host, f.opts.Domains); matched != "" {
			f.rewrite(matched, req)
		}
	}
}

func (f *ConfigServerFilter) rewrite(host string, req *http.Request) {
	req.URL.Scheme = "https"
	prevHost := req.Host
	req.Host = f.fromDNSCache(host) + ":443"
	req.Header.Set(common.CfgSvrAuthTokenHeader, f.opts.AuthToken)
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return
	}
	req.Header.Set(common.CfgSvrClientIPHeader, ip)
	log.Debugf("Rewrote request from %s to %s as \"GET %s\", host %s", ip, prevHost, req.URL.String(), req.Host)
}

// in returns the host portion if it's is in the domains list, or returns ""
func in(hostport string, domains []string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	for _, d := range domains {
		if host == d {
			return d
		}
	}
	return ""
}

func (f *ConfigServerFilter) fromDNSCache(host string) string {
	f.dnsCacheMutex.RLock()
	defer f.dnsCacheMutex.RUnlock()
	resolved, ok := f.dnsCache[host]
	if ok {
		return resolved
	}
	// If for some odd reason we can't find the host in the cache, just ignore the cache and return
	// the host
	log.Errorf("CACHE MISS FOR %v", host)
	return host
}

func (f *ConfigServerFilter) resolveDomain(domain string) string {
	addrs, err := net.LookupHost(domain)
	if err != nil {
		log.Errorf("Could not lookup %v", domain)
		return domain
	}
	if len(addrs) == 0 {
		return domain
	}
	addr := addrs[f.random.Intn(len(addrs))]
	log.Debugf("Resolved addr %v", addr)
	return addr
}
