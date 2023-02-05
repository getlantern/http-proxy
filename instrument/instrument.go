package instrument

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
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
	ProxiedBytes(ctx context.Context, sent, recv int, platform, version, app, locale, dataCapCohort string, clientIP net.IP, deviceID, originHost string)
	ReportToOTELPeriodically(interval time.Duration, tp *sdktrace.TracerProvider, includeDeviceID bool)
	ReportToOTEL(tp *sdktrace.TracerProvider, includeDeviceID bool)
	quicSentPacket(ctx context.Context)
	quicLostPacket(ctx context.Context)
}

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
	for _, _ = range protocols {
		trackers = append(trackers, multipath.NullTracker{})
	}
	return
}
func (i NoInstrument) Throttle(ctx context.Context, m bool, reason string) {}

func (i NoInstrument) XBQHeaderSent(ctx context.Context)                                      {}
func (i NoInstrument) SuspectedProbing(ctx context.Context, fromIP net.IP, reason string)     {}
func (i NoInstrument) VersionCheck(ctx context.Context, redirect bool, method, reason string) {}
func (i NoInstrument) ProxiedBytes(ctx context.Context, sent, recv int, platform, version, app, locale, dataCapCohort string, clientIP net.IP, deviceID, originHost string) {
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

// PromLabels turns the common labels to Prometheus form.
func (c *CommonLabels) PromLabels() prometheus.Labels {
	return map[string]string{
		"protocol":                c.Protocol,
		"build_type":              c.BuildType,
		"support_tls_resumption":  strconv.FormatBool(c.SupportTLSResumption),
		"require_tls_resumption":  strconv.FormatBool(c.RequireTLSResumption),
		"missing_ticket_reaction": c.MissingTicketReaction,
	}
}

type instrumentedFilter struct {
	requests prometheus.Counter
	errors   prometheus.Counter
	duration prometheus.Observer
	filters.Filter
}

func (f *instrumentedFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	start := time.Now()
	res, cs, err := f.Filter.Apply(cs, req, next)
	f.requests.Inc()
	if err != nil {
		f.errors.Inc()
	}
	f.duration.Observe(time.Since(start).Seconds())
	return res, cs, err
}

// prominstrument is an implementation of Instrument which exports Prometheus
// metrics.
type prominstrument struct {
	countryLookup           geo.CountryLookup
	ispLookup               geo.ISPLookup
	commonLabels            prometheus.Labels
	commonLabelNames        []string
	filters                 map[string]filters.Filter
	errorHandlers           map[string]func(conn net.Conn, err error)
	clientStats             map[clientDetails]*usage
	clientStatsWithDeviceID map[clientDetails]*usage
	originStats             map[originDetails]*usage
	statsMx                 sync.Mutex

	blacklistChecked, blacklisted, mimicryChecked, mimicked, quicLostPackets, quicSentPackets, throttlingChecked, xbqSent prometheus.Counter

	bytesSent, bytesRecv, bytesSentByISP, bytesRecvByISP, throttled, notThrottled, suspectedProbing, versionCheck *prometheus.CounterVec

	mpFramesSent, mpBytesSent, mpFramesReceived, mpBytesReceived, mpFramesRetransmitted, mpBytesRetransmitted *prometheus.CounterVec
}

func NewPrometheus(countryLookup geo.CountryLookup, ispLookup geo.ISPLookup, c CommonLabels) (*prominstrument, error) {
	if err := otelinstrument.Initialize(); err != nil {
		return nil, err
	}

	commonLabels := c.PromLabels()
	commonLabelNames := make([]string, len(commonLabels))
	i := 0
	for k := range commonLabels {
		commonLabelNames[i] = k
		i++
	}
	p := &prominstrument{
		countryLookup:           countryLookup,
		ispLookup:               ispLookup,
		commonLabels:            commonLabels,
		commonLabelNames:        commonLabelNames,
		filters:                 make(map[string]filters.Filter),
		errorHandlers:           make(map[string]func(conn net.Conn, err error)),
		clientStats:             make(map[clientDetails]*usage),
		clientStatsWithDeviceID: make(map[clientDetails]*usage),
		originStats:             make(map[originDetails]*usage),
		blacklistChecked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_blacklist_checked_requests_total",
		}, commonLabelNames).With(commonLabels),
		blacklisted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_blacklist_blacklisted_requests_total",
		}, commonLabelNames).With(commonLabels),
		bytesSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_downstream_sent_bytes_total",
			Help: "Bytes sent to the client connections. Pluggable transport overhead excluded",
		}, append(commonLabelNames, "app_platform", "app_version", "app", "datacap_cohort")).MustCurryWith(commonLabels),
		bytesRecv: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_downstream_received_bytes_total",
			Help: "Bytes received from the client connections. Pluggable transport overhead excluded",
		}, append(commonLabelNames, "app_platform", "app_version", "app", "datacap_cohort")).MustCurryWith(commonLabels),
		bytesSentByISP: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_downstream_by_isp_sent_bytes_total",
			Help: "Bytes sent to the client connections, by country and isp. Pluggable transport overhead excluded",
		}, append(commonLabelNames, "country", "isp")).MustCurryWith(commonLabels),
		bytesRecvByISP: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_downstream_by_isp_received_bytes_total",
			Help: "Bytes received from the client connections, by country and isp. Pluggable transport overhead excluded",
		}, append(commonLabelNames, "country", "isp")).MustCurryWith(commonLabels),

		quicLostPackets: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_downstream_quic_lost_packets_total",
			Help: "Number of QUIC packets lost and effectively resent to the client connections.",
		}, commonLabelNames).With(commonLabels),
		quicSentPackets: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_downstream_quic_sent_packets_total",
			Help: "Number of QUIC packets sent to the client connections.",
		}, commonLabelNames).With(commonLabels),

		mimicryChecked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_apache_mimicry_checked_total",
		}, commonLabelNames).With(commonLabels),
		mimicked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_apache_mimicry_mimicked_total",
		}, commonLabelNames).With(commonLabels),

		mpFramesSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_multipath_sent_frames_total",
		}, append(commonLabelNames, "path_protocol")).MustCurryWith(commonLabels),
		mpBytesSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_multipath_sent_bytes_total",
		}, append(commonLabelNames, "path_protocol")).MustCurryWith(commonLabels),
		mpFramesReceived: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_multipath_received_frames_total",
		}, append(commonLabelNames, "path_protocol")).MustCurryWith(commonLabels),
		mpBytesReceived: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_multipath_received_bytes_total",
		}, append(commonLabelNames, "path_protocol")).MustCurryWith(commonLabels),
		mpFramesRetransmitted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_multipath_retransmissions_total",
		}, append(commonLabelNames, "path_protocol")).MustCurryWith(commonLabels),
		mpBytesRetransmitted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_multipath_retransmission_bytes_total",
		}, append(commonLabelNames, "path_protocol")).MustCurryWith(commonLabels),

		xbqSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_xbq_header_sent_total",
		}, commonLabelNames).With(commonLabels),

		throttlingChecked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_device_throttling_checked_total",
		}, commonLabelNames).With(commonLabels),
		throttled: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_device_throttling_throttled_total",
		}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels),
		notThrottled: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_device_throttling_not_throttled_total",
		}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels),

		suspectedProbing: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_suspected_probing_total",
		}, append(commonLabelNames, "country", "reason")).MustCurryWith(commonLabels),

		versionCheck: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_version_check_total",
		}, append(commonLabelNames, "method", "redirected", "reason")).MustCurryWith(commonLabels),
	}

	return p, nil
}

// Run runs the PromInstrument exporter on the given address. The
// path is /metrics.
func (p *prominstrument) Run(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server.ListenAndServe()
}

// WrapFilter wraps a filter to instrument the requests/errors/duration
// (so-called RED) of processed requests.
func (p *prominstrument) WrapFilter(prefix string, f filters.Filter) (filters.Filter, error) {
	wrapped := p.filters[prefix]
	if wrapped == nil {
		wrapped = &instrumentedFilter{
			promauto.NewCounterVec(prometheus.CounterOpts{
				Name: prefix + "_requests_total",
			}, p.commonLabelNames).With(p.commonLabels),
			promauto.NewCounterVec(prometheus.CounterOpts{
				Name: prefix + "_request_errors_total",
			}, p.commonLabelNames).With(p.commonLabels),
			promauto.NewHistogramVec(prometheus.HistogramOpts{
				Name:    prefix + "_request_duration_seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1},
			}, p.commonLabelNames).With(p.commonLabels),
			f}
		p.filters[prefix] = wrapped
	}

	var err error
	wrapped, err = otelinstrument.WrapFilter(prefix, wrapped)
	if err != nil {
		return nil, err
	}
	return wrapped, nil
}

// WrapConnErrorHandler wraps an error handler to instrument the error count.
func (p *prominstrument) WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) (func(conn net.Conn, err error), error) {
	h := p.errorHandlers[prefix]
	if h == nil {
		errors := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: prefix + "_errors_total",
		}, p.commonLabelNames).With(p.commonLabels)
		consec_errors := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: prefix + "_consec_per_client_ip_errors_total",
		}, p.commonLabelNames).With(p.commonLabels)
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
			errors.Inc()
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
				consec_errors.Inc()
				otelConsecCounter.Add(context.Background(), 1)
			} else {
				mu.Unlock()
			}
			f(conn, err)
		}
		p.errorHandlers[prefix] = h
	}
	return h, nil
}

// Blacklist instruments the blacklist checking.
func (p *prominstrument) Blacklist(ctx context.Context, b bool) {
	otelinstrument.Blacklist.Add(ctx, 1,
		attribute.KeyValue{"blacklisted", attribute.BoolValue(b)})

	p.blacklistChecked.Inc()
	if b {
		p.blacklisted.Inc()
	}
}

// Mimic instruments the Apache mimicry.
func (p *prominstrument) Mimic(ctx context.Context, m bool) {
	otelinstrument.Mimicked.Add(ctx, 1, attribute.KeyValue{"mimicked", attribute.BoolValue(m)})

	p.mimicryChecked.Inc()
	if m {
		p.mimicked.Inc()
		otelinstrument.Mimicked.Add(ctx, 1)
	}
}

// Throttle instruments the device based throttling.
func (p *prominstrument) Throttle(ctx context.Context, m bool, reason string) {
	p.throttlingChecked.Inc()
	otelinstrument.Throttling.Add(ctx, 1,
		attribute.KeyValue{"throttled", attribute.BoolValue(m)},
		attribute.KeyValue{"reason", attribute.StringValue(reason)})

	if m {
		p.throttled.With(prometheus.Labels{"reason": reason}).Inc()
	} else {
		p.notThrottled.With(prometheus.Labels{"reason": reason}).Inc()
	}
}

// XBQHeaderSent counts the number of times XBQ header is sent along with the
// response.
func (p *prominstrument) XBQHeaderSent(ctx context.Context) {
	p.xbqSent.Inc()
	otelinstrument.XBQ.Add(ctx, 1)
}

// SuspectedProbing records the number of visits which looks like active
// probing.
func (p *prominstrument) SuspectedProbing(ctx context.Context, fromIP net.IP, reason string) {
	fromCountry := p.countryLookup.CountryCode(fromIP)
	p.suspectedProbing.With(prometheus.Labels{"country": fromCountry, "reason": reason}).Inc()
	otelinstrument.SuspectedProbing.Add(
		ctx,
		1,
		attribute.KeyValue{"country", attribute.StringValue(fromCountry)},
		attribute.KeyValue{"reason", attribute.StringValue(reason)},
	)
}

// VersionCheck records the number of times the Lantern version header is
// checked and if redirecting to the upgrade page is required.
func (p *prominstrument) VersionCheck(ctx context.Context, redirect bool, method, reason string) {
	labels := prometheus.Labels{"method": method, "redirected": strconv.FormatBool(redirect), "reason": reason}
	p.versionCheck.With(labels).Inc()
	otelinstrument.VersionCheck.Add(
		ctx,
		1,
		attribute.KeyValue{"method", attribute.StringValue(method)},
		attribute.KeyValue{"redirected", attribute.BoolValue(redirect)},
		attribute.KeyValue{"reason", attribute.StringValue(reason)},
	)
}

// ProxiedBytes records the volume of application data clients sent and
// received via the proxy.
func (p *prominstrument) ProxiedBytes(ctx context.Context, sent, recv int, platform, version, app, locale, dataCapCohort string, clientIP net.IP, deviceID, originHost string) {
	labels := prometheus.Labels{"app_platform": platform, "app_version": version, "app": app, "datacap_cohort": dataCapCohort}
	p.bytesSent.With(labels).Add(float64(sent))
	p.bytesRecv.With(labels).Add(float64(recv))

	// Track the cardinality of clients.
	otelinstrument.DistinctClients1m.Add(deviceID)
	otelinstrument.DistinctClients10m.Add(deviceID)
	otelinstrument.DistinctClients1h.Add(deviceID)

	country := p.countryLookup.CountryCode(clientIP)
	isp := p.ispLookup.ISP(clientIP)
	asn := p.ispLookup.ASN(clientIP)
	by_isp := prometheus.Labels{"country": country, "isp": "omitted"}
	// We care about ISPs within these countries only, to reduce cardinality of the metrics
	if country == "CN" || country == "IR" || country == "AE" || country == "TK" {
		by_isp["isp"] = isp
	}
	p.bytesSentByISP.With(by_isp).Add(float64(sent))
	p.bytesRecvByISP.With(by_isp).Add(float64(recv))
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
		append(otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("transmit")})...,
	)

	otelinstrument.ProxyIO.Add(
		ctx,
		int64(sent),
		append(otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("receive")})...,
	)

	clientKey := clientDetails{
		platform: platform,
		version:  version,
		locale:   locale,
		country:  country,
		isp:      isp,
		asn:      asn,
	}
	clientKeyWithDeviceID := clientDetails{
		deviceID: deviceID,
		platform: platform,
		version:  version,
		locale:   locale,
		country:  country,
		isp:      isp,
		asn:      asn,
	}
	p.statsMx.Lock()
	p.clientStats[clientKey] = p.clientStats[clientKey].add(sent, recv)
	p.clientStatsWithDeviceID[clientKeyWithDeviceID] = p.clientStatsWithDeviceID[clientKeyWithDeviceID].add(sent, recv)
	if originHost != "" {
		originRoot, err := p.originRoot(originHost)
		if err == nil {
			// only record if we could extract originRoot
			originKey := originDetails{
				origin:   originRoot,
				platform: platform,
				version:  version,
				country:  country,
			}
			p.originStats[originKey] = p.originStats[originKey].add(sent, recv)
		}
	}
	p.statsMx.Unlock()
}

// quicPackets is used by QuicTracer to update QUIC retransmissions mainly for block detection.
func (p *prominstrument) quicSentPacket(ctx context.Context) {
	p.quicSentPackets.Inc()
	otelinstrument.QuicPackets.Add(ctx, 1, attribute.KeyValue{"state", attribute.StringValue("sent")})
}

func (p *prominstrument) quicLostPacket(ctx context.Context) {
	p.quicLostPackets.Inc()
	otelinstrument.QuicPackets.Add(ctx, 1, attribute.KeyValue{"state", attribute.StringValue("lost")})
}

type stats struct {
	otelAttributes      []attribute.KeyValue
	framesSent          prometheus.Counter
	bytesSent           prometheus.Counter
	framesRetransmitted prometheus.Counter
	bytesRetransmitted  prometheus.Counter
	framesReceived      prometheus.Counter
	bytesReceived       prometheus.Counter
}

func (s *stats) OnRecv(n uint64) {
	s.framesReceived.Inc()
	s.bytesReceived.Add(float64(n))
	otelinstrument.MultipathFrames.Add(context.Background(), 1,
		append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("receive")})...)
	otelinstrument.MultipathIO.Add(context.Background(), int64(n),
		append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("receive")})...)
}
func (s *stats) OnSent(n uint64) {
	s.framesSent.Inc()
	s.bytesSent.Add(float64(n))
	otelinstrument.MultipathFrames.Add(context.Background(), 1,
		append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("transmit")})...)
	otelinstrument.MultipathIO.Add(context.Background(), int64(n),
		append(s.otelAttributes, attribute.KeyValue{"direction", attribute.StringValue("transmit")})...)
}
func (s *stats) OnRetransmit(n uint64) {
	s.framesRetransmitted.Inc()
	s.bytesRetransmitted.Add(float64(n))
	otelinstrument.MultipathFrames.Add(context.Background(), 1,
		append(s.otelAttributes,
			attribute.KeyValue{"direction", attribute.StringValue("transmit")},
			attribute.KeyValue{"state", attribute.StringValue("retransmit")})...)
	otelinstrument.MultipathIO.Add(context.Background(), int64(n),
		append(s.otelAttributes,
			attribute.KeyValue{"direction", attribute.StringValue("transmit")},
			attribute.KeyValue{"state", attribute.StringValue("retransmit")})...)
}
func (s *stats) UpdateRTT(time.Duration) {
	// do nothing as the RTT from different clients can vary significantly
}

func (prom *prominstrument) MultipathStats(protocols []string) (trackers []multipath.StatsTracker) {
	for _, p := range protocols {
		path_protocol := prometheus.Labels{"path_protocol": p}
		trackers = append(trackers, &stats{
			framesSent:          prom.mpFramesSent.With(path_protocol),
			bytesSent:           prom.mpBytesSent.With(path_protocol),
			framesReceived:      prom.mpFramesReceived.With(path_protocol),
			bytesReceived:       prom.mpBytesReceived.With(path_protocol),
			framesRetransmitted: prom.mpFramesRetransmitted.With(path_protocol),
			bytesRetransmitted:  prom.mpBytesRetransmitted.With(path_protocol),
			otelAttributes:      []attribute.KeyValue{attribute.KeyValue{"path_protocol", attribute.StringValue(p)}},
		})
	}
	return
}

type clientDetails struct {
	deviceID string
	platform string
	version  string
	locale   string
	country  string
	isp      string
	asn      string
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

func (p *prominstrument) ReportToOTELPeriodically(interval time.Duration, tp *sdktrace.TracerProvider, includeDeviceID bool) {
	for {
		// We randomize the sleep time to avoid bursty submission to OpenTelemetry.
		// Even though each proxy sends relatively little data, proxies often run fairly
		// closely synchronized since they all update to a new binary and restart around the same
		// time. By randomizing each proxy's interval, we smooth out the pattern of submissions.
		sleepInterval := rand.Int63n(int64(interval * 2))
		time.Sleep(time.Duration(sleepInterval))
		p.ReportToOTEL(tp, includeDeviceID)
	}
}

func (p *prominstrument) ReportToOTEL(tp *sdktrace.TracerProvider, includeDeviceID bool) {
	var clientStats map[clientDetails]*usage
	p.statsMx.Lock()
	if includeDeviceID {
		clientStats = p.clientStatsWithDeviceID
		p.clientStatsWithDeviceID = make(map[clientDetails]*usage)
	} else {
		clientStats = p.clientStats
		p.clientStats = make(map[clientDetails]*usage)
	}
	originStats := p.originStats
	p.originStats = make(map[originDetails]*usage)
	p.statsMx.Unlock()

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
					attribute.String("client_asn", key.asn)))
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

func (p *prominstrument) originRoot(origin string) (string, error) {
	ip := net.ParseIP(origin)
	if ip != nil {
		// origin is an IP address, try to get domain name
		origins, err := net.LookupAddr(origin)
		if err != nil || net.ParseIP(origins[0]) != nil {
			// failed to reverse lookup, try to get ASN
			asn := p.ispLookup.ASN(ip)
			if asn != "" {
				return asn, nil
			}
			return "", errors.New("unable to lookup ip %v", ip)
		}
		return p.originRoot(stripTrailingDot(origins[0]))
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
