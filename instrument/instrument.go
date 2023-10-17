package instrument

import (
	"context"
	"math/rand"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/getlantern/errors"
	"github.com/getlantern/geo"
	"github.com/getlantern/http-proxy-lantern/v2/instrument/otelinstrument"
	"github.com/getlantern/multipath"
	"github.com/getlantern/proxy/v2/filters"
)

var (
	originRootRegex = regexp.MustCompile(`([^\.]+\.[^\.]+$)`)
)

// Instrument is the common interface about what can be instrumented.
type Instrument interface {
	WrapFilter(prefix string, f filters.Filter) (filters.Filter, error)
	WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) (func(conn net.Conn, err error), error)
	Blacklist(ctx context.Context, b bool)
	Mimic(ctx context.Context, m bool)
	MultipathStats([]string) []multipath.StatsTracker
	Throttle(ctx context.Context, m bool, reason string)
	XBQHeaderSent(ctx context.Context)
	SuspectedProbing(ctx context.Context, fromIP net.IP, reason string)
	VersionCheck(ctx context.Context, redirect bool, method, reason string)
	ProxiedBytes(ctx context.Context, sent, recv int, platform, version, app, locale, dataCapCohort, probingError string, clientIP net.IP, deviceID, originHost string)
	ReportToOTELPeriodically(interval time.Duration, tp *sdktrace.TracerProvider, includeDeviceID bool)
	ReportToOTEL(tp *sdktrace.TracerProvider, includeDeviceID bool)
	quicSentPacket(ctx context.Context)
	quicLostPacket(ctx context.Context)
}

var _ Instrument = &NoInstrument{}
var _ Instrument = &defaultInstrument{}

// NoInstrument is an implementation of Instrument which does nothing
type NoInstrument struct {
}

func (i NoInstrument) WrapFilter(prefix string, f filters.Filter) (filters.Filter, error) {
	return f, nil
}
func (i NoInstrument) WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) (func(conn net.Conn, err error), error) {
	return f, nil
}
func (i NoInstrument) Blacklist(ctx context.Context, b bool) {}
func (i NoInstrument) Mimic(ctx context.Context, m bool)     {}
func (i NoInstrument) MultipathStats(protocols []string) (trackers []multipath.StatsTracker) {
	for range protocols {
		trackers = append(trackers, multipath.NullTracker{})
	}
	return
}
func (i NoInstrument) Throttle(ctx context.Context, m bool, reason string) {}

func (i NoInstrument) XBQHeaderSent(ctx context.Context)                                      {}
func (i NoInstrument) SuspectedProbing(ctx context.Context, fromIP net.IP, reason string)     {}
func (i NoInstrument) VersionCheck(ctx context.Context, redirect bool, method, reason string) {}
func (i NoInstrument) ProxiedBytes(ctx context.Context, sent, recv int, platform, version, app, locale, dataCapCohort, probingError string, clientIP net.IP, deviceID, originHost string) {
}
func (i NoInstrument) ReportToOTELPeriodically(interval time.Duration, tp *sdktrace.TracerProvider, includeDeviceID bool) {
}
func (i NoInstrument) ReportToOTEL(tp *sdktrace.TracerProvider, includeDeviceID bool) {
}
func (i NoInstrument) quicSentPacket(ctx context.Context) {}
func (i NoInstrument) quicLostPacket(ctx context.Context) {}

// CommonLabels defines a set of common labels apply to all metrics instrumented.
type CommonLabels struct {
	Protocol              string
	BuildType             string
	SupportTLSResumption  bool
	RequireTLSResumption  bool
	MissingTicketReaction string
}

// defaultInstrument is an implementation of Instrument which exports metrics
// via open telemetry.
type defaultInstrument struct {
	countryLookup           geo.CountryLookup
	ispLookup               geo.ISPLookup
	filters                 map[string]filters.Filter
	errorHandlers           map[string]func(conn net.Conn, err error)
	clientStats             map[clientDetails]*usage
	clientStatsWithDeviceID map[clientDetails]*usage
	originStats             map[originDetails]*usage
	statsMx                 sync.Mutex
}

func NewDefault(countryLookup geo.CountryLookup, ispLookup geo.ISPLookup) (*defaultInstrument, error) {
	if err := otelinstrument.Initialize(); err != nil {
		return nil, err
	}

	p := &defaultInstrument{
		countryLookup:           countryLookup,
		ispLookup:               ispLookup,
		filters:                 make(map[string]filters.Filter),
		errorHandlers:           make(map[string]func(conn net.Conn, err error)),
		clientStats:             make(map[clientDetails]*usage),
		clientStatsWithDeviceID: make(map[clientDetails]*usage),
		originStats:             make(map[originDetails]*usage),
	}

	return p, nil
}

// WrapFilter wraps a filter to instrument the requests/errors/duration
// (so-called RED) of processed requests.
func (ins *defaultInstrument) WrapFilter(prefix string, f filters.Filter) (filters.Filter, error) {
	wrapped := ins.filters[prefix]
	if wrapped == nil {
		var err error
		wrapped, err = otelinstrument.WrapFilter(prefix, f)
		if err != nil {
			return nil, err
		}
		ins.filters[prefix] = wrapped
	}
	return wrapped, nil
}

// WrapConnErrorHandler wraps an error handler to instrument the error count.
func (ins *defaultInstrument) WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) (func(conn net.Conn, err error), error) {
	h := ins.errorHandlers[prefix]
	if h == nil {
		otelCounter, err := otelinstrument.ConnErrorHandlerCounter(prefix)
		if err != nil {
			return nil, err
		}
		otelConsecCounter, err := otelinstrument.ConnConsecErrorHandlerCounter(prefix)
		if err != nil {
			return nil, err
		}
		if f == nil {
			f = func(conn net.Conn, err error) {}
		}
		var mu sync.Mutex
		var lastRemoteIP string
		h = func(conn net.Conn, err error) {
			otelCounter.Add(context.Background(), 1)
			addr := conn.RemoteAddr()
			if addr == nil {
				return
			}
			host, _, err := net.SplitHostPort(addr.String())
			if err != nil {
				return
			}
			mu.Lock()
			if lastRemoteIP != host {
				lastRemoteIP = host
				mu.Unlock()
				otelConsecCounter.Add(context.Background(), 1)
			} else {
				mu.Unlock()
			}
			f(conn, err)
		}
		ins.errorHandlers[prefix] = h
	}
	return h, nil
}

// Blacklist instruments the blacklist checking.
func (ins *defaultInstrument) Blacklist(ctx context.Context, b bool) {
	otelinstrument.Blacklist.Add(ctx, 1,
		metric.WithAttributes(attribute.KeyValue{"blacklisted", attribute.BoolValue(b)}))
}

// Mimic instruments the Apache mimicry.
func (ins *defaultInstrument) Mimic(ctx context.Context, m bool) {
	otelinstrument.Mimicked.Add(ctx, 1, metric.WithAttributes(attribute.KeyValue{"mimicked", attribute.BoolValue(m)}))

	if m {
		otelinstrument.Mimicked.Add(ctx, 1)
	}
}

// Throttle instruments the device based throttling.
func (ins *defaultInstrument) Throttle(ctx context.Context, m bool, reason string) {
	otelinstrument.Throttling.Add(ctx, 1,
		metric.WithAttributes(
			attribute.KeyValue{"throttled", attribute.BoolValue(m)},
			attribute.KeyValue{"reason", attribute.StringValue(reason)},
		))
}

// XBQHeaderSent counts the number of times XBQ header is sent along with the
// response.
func (ins *defaultInstrument) XBQHeaderSent(ctx context.Context) {
	otelinstrument.XBQ.Add(ctx, 1)
}

// SuspectedProbing records the number of visits which looks like active
// probing.
func (ins *defaultInstrument) SuspectedProbing(ctx context.Context, fromIP net.IP, reason string) {
	fromCountry := ins.countryLookup.CountryCode(fromIP)
	otelinstrument.SuspectedProbing.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.KeyValue{"country", attribute.StringValue(fromCountry)},
			attribute.KeyValue{"reason", attribute.StringValue(reason)},
		),
	)
}

// VersionCheck records the number of times the Lantern version header is
// checked and if redirecting to the upgrade page is required.
func (ins *defaultInstrument) VersionCheck(ctx context.Context, redirect bool, method, reason string) {
	otelinstrument.VersionCheck.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.KeyValue{"method", attribute.StringValue(method)},
			attribute.KeyValue{"redirected", attribute.BoolValue(redirect)},
			attribute.KeyValue{"reason", attribute.StringValue(reason)},
		),
	)
}

// ProxiedBytes records the volume of application data clients sent and
// received via the proxy.
func (ins *defaultInstrument) ProxiedBytes(ctx context.Context, sent, recv int, platform, version, app, locale, dataCapCohort, probingError string, clientIP net.IP, deviceID, originHost string) {
	// Track the cardinality of clients.
	otelinstrument.DistinctClients1m.Add(deviceID)
	otelinstrument.DistinctClients10m.Add(deviceID)
	otelinstrument.DistinctClients1h.Add(deviceID)

	country := ins.countryLookup.CountryCode(clientIP)
	isp := ins.ispLookup.ISP(clientIP)
	asn := ins.ispLookup.ASN(clientIP)
	otelAttributes := []attribute.KeyValue{
		{"client_platform", attribute.StringValue(platform)},
		{"client_version", attribute.StringValue(version)},
		{"client_app", attribute.StringValue(app)},
		{"datacap_cohort", attribute.StringValue(dataCapCohort)},
		{"country", attribute.StringValue(country)},
		{"client_isp", attribute.StringValue(isp)},
		{"client_asn", attribute.StringValue(asn)},
	}

	otelinstrument.ProxyIO.Add(
		ctx,
		int64(sent),
		metric.WithAttributes(
			append(otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("transmit")})...,
		),
	)

	otelinstrument.ProxyIO.Add(
		ctx,
		int64(recv),
		metric.WithAttributes(
			append(otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("receive")})...,
		),
	)

	clientKey := clientDetails{
		platform:     platform,
		version:      version,
		locale:       locale,
		country:      country,
		isp:          isp,
		asn:          asn,
		probingError: probingError,
	}
	clientKeyWithDeviceID := clientDetails{
		deviceID:     deviceID,
		platform:     platform,
		version:      version,
		locale:       locale,
		country:      country,
		isp:          isp,
		asn:          asn,
		probingError: probingError,
	}

	var originKey originDetails
	hasOriginKey := true
	if originHost != "" {
		originRoot, err := ins.originRoot(originHost)
		if err == nil {
			// only record if we could extract originRoot
			originKey = originDetails{
				origin:   originRoot,
				platform: platform,
				version:  version,
				country:  country,
			}
			hasOriginKey = true
		}
	}

	ins.statsMx.Lock()
	ins.clientStats[clientKey] = ins.clientStats[clientKey].add(sent, recv)
	ins.clientStatsWithDeviceID[clientKeyWithDeviceID] = ins.clientStatsWithDeviceID[clientKeyWithDeviceID].add(sent, recv)
	if hasOriginKey {
		ins.originStats[originKey] = ins.originStats[originKey].add(sent, recv)
	}
	ins.statsMx.Unlock()
}

// quicPackets is used by QuicTracer to update QUIC retransmissions mainly for block detection.
func (ins *defaultInstrument) quicSentPacket(ctx context.Context) {
	otelinstrument.QuicPackets.Add(ctx, 1, metric.WithAttributes(attribute.KeyValue{"state", attribute.StringValue("sent")}))
}

func (ins *defaultInstrument) quicLostPacket(ctx context.Context) {
	otelinstrument.QuicPackets.Add(ctx, 1, metric.WithAttributes(attribute.KeyValue{"state", attribute.StringValue("lost")}))
}

type stats struct {
	otelAttributes []attribute.KeyValue
}

func (s *stats) OnRecv(n uint64) {
	otelinstrument.MultipathFrames.Add(context.Background(), 1,
		metric.WithAttributes(append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("receive")})...))
	otelinstrument.MultipathIO.Add(context.Background(), int64(n),
		metric.WithAttributes(append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("receive")})...))
}
func (s *stats) OnSent(n uint64) {
	otelinstrument.MultipathFrames.Add(context.Background(), 1,
		metric.WithAttributes(append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("transmit")})...))
	otelinstrument.MultipathIO.Add(context.Background(), int64(n),
		metric.WithAttributes(append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("transmit")})...))
}
func (s *stats) OnRetransmit(n uint64) {
	otelinstrument.MultipathFrames.Add(context.Background(), 1,
		metric.WithAttributes(append(s.otelAttributes,
			attribute.KeyValue{"direction", attribute.StringValue("transmit")},
			attribute.KeyValue{"state", attribute.StringValue("retransmit")})...))
	otelinstrument.MultipathIO.Add(context.Background(), int64(n),
		metric.WithAttributes(append(s.otelAttributes,
			attribute.KeyValue{"direction", attribute.StringValue("transmit")},
			attribute.KeyValue{"state", attribute.StringValue("retransmit")})...))
}
func (s *stats) UpdateRTT(time.Duration) {
	// do nothing as the RTT from different clients can vary significantly
}

func (ins *defaultInstrument) MultipathStats(protocols []string) (trackers []multipath.StatsTracker) {
	for _, p := range protocols {
		trackers = append(trackers, &stats{
			otelAttributes: []attribute.KeyValue{attribute.KeyValue{"path_protocol", attribute.StringValue(p)}},
		})
	}
	return
}

type clientDetails struct {
	deviceID     string
	platform     string
	version      string
	locale       string
	country      string
	isp          string
	asn          string
	probingError string
}

type originDetails struct {
	origin   string
	platform string
	version  string
	country  string
}

type usage struct {
	sent int
	recv int
}

func (u *usage) add(sent int, recv int) *usage {
	if u == nil {
		u = &usage{}
	}
	u.sent += sent
	u.recv += recv
	return u
}

func (ins *defaultInstrument) ReportToOTELPeriodically(interval time.Duration, tp *sdktrace.TracerProvider, includeDeviceID bool) {
	for {
		// We randomize the sleep time to avoid bursty submission to OpenTelemetry.
		// Even though each proxy sends relatively little data, proxies often run fairly
		// closely synchronized since they all update to a new binary and restart around the same
		// time. By randomizing each proxy's interval, we smooth out the pattern of submissions.
		sleepInterval := rand.Int63n(int64(interval * 2))
		time.Sleep(time.Duration(sleepInterval))
		ins.ReportToOTEL(tp, includeDeviceID)
	}
}

func (ins *defaultInstrument) ReportToOTEL(tp *sdktrace.TracerProvider, includeDeviceID bool) {
	var clientStats map[clientDetails]*usage
	ins.statsMx.Lock()
	if includeDeviceID {
		clientStats = ins.clientStatsWithDeviceID
		ins.clientStatsWithDeviceID = make(map[clientDetails]*usage)
	} else {
		clientStats = ins.clientStats
		ins.clientStats = make(map[clientDetails]*usage)
	}
	originStats := ins.originStats
	ins.originStats = make(map[originDetails]*usage)
	ins.statsMx.Unlock()

	for key, value := range clientStats {
		_, span := tp.Tracer("").
			Start(
				context.Background(),
				"proxied_bytes",
				trace.WithAttributes(
					attribute.Int("bytes_sent", value.sent),
					attribute.Int("bytes_recv", value.recv),
					attribute.Int("bytes_total", value.sent+value.recv),
					attribute.String("device_id", key.deviceID),
					attribute.String("client_platform", key.platform),
					attribute.String("client_version", key.version),
					attribute.String("client_locale", key.locale),
					attribute.String("client_country", key.country),
					attribute.String("client_isp", key.isp),
					attribute.String("client_asn", key.asn),
					attribute.String("probing_error", key.probingError)))
		span.End()
	}
	if !includeDeviceID {
		// In order to prevent associating origins with device IDs, only report
		// origin stats if we're not including device IDs.
		for key, value := range originStats {
			_, span := tp.Tracer("").
				Start(
					context.Background(),
					"origin_bytes",
					trace.WithAttributes(
						attribute.Int("origin_bytes_sent", value.sent),
						attribute.Int("origin_bytes_recv", value.recv),
						attribute.Int("origin_bytes_total", value.sent+value.recv),
						attribute.String("origin", key.origin),
						attribute.String("client_platform", key.platform),
						attribute.String("client_version", key.version),
						attribute.String("client_country", key.country)))
			span.End()
		}
	}
}

func (ins *defaultInstrument) originRoot(origin string) (string, error) {
	ip := net.ParseIP(origin)
	if ip != nil {
		var r net.Resolver
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// origin is an IP address, try to get domain name
		origins, err := r.LookupAddr(ctx, origin)
		if err != nil || net.ParseIP(origins[0]) != nil {
			// failed to reverse lookup, try to get ASN
			asn := ins.ispLookup.ASN(ip)
			if asn != "" {
				return asn, nil
			}
			return "", errors.New("unable to lookup ip %v", ip)
		}
		return ins.originRoot(stripTrailingDot(origins[0]))
	}
	matches := originRootRegex.FindStringSubmatch(origin)
	if matches == nil {
		// regex didn't match, return origin as is
		return origin, nil
	}
	return matches[1], nil
}

func stripTrailingDot(s string) string {
	return strings.TrimRight(s, ".")
}
