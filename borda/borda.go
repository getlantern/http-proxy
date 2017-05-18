package borda

import (
	"crypto/tls"
	"math/rand"
	"net"
	"time"

	borda "github.com/getlantern/borda/client"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
	"github.com/getlantern/ops"
	"github.com/getlantern/zenodb/rpc"
)

var (
	log = golog.LoggerFor("lantern-proxy-borda")

	fullyReportedOps = []string{"tcpinfo", "google_search", "google_captcha"}
)

// Enable enables borda reporting
func Enable(bordaReportInterval time.Duration, bordaSamplePercentage float64, maxBufferSize int) listeners.MeasuredReportFN {
	inSample := func(ctx map[string]interface{}) bool {
		if rand.Float64() < bordaSamplePercentage {
			return true
		}

		// For some ops, we don't randomly sample, we include all of them
		op := ctx["op"]
		switch t := op.(type) {
		case string:
			for _, fullyReportedOp := range fullyReportedOps {
				if t == fullyReportedOp {
					log.Tracef("Including fully reported op %v in borda sample", fullyReportedOp)
					return true
				}
			}
			return false
		default:
			return false
		}
	}

	opts := &borda.Options{
		BatchInterval: bordaReportInterval,
	}

	clientSessionCache := tls.NewLRUClientSessionCache(10000)
	clientTLSConfig := &tls.Config{
		ServerName:         "borda.getlantern.org",
		ClientSessionCache: clientSessionCache,
	}

	rc, err := rpc.Dial("borda.getlantern.org:17712", &rpc.ClientOpts{
		Dialer: func(addr string, timeout time.Duration) (net.Conn, error) {
			conn, dialErr := net.DialTimeout("tcp", addr, timeout)
			if dialErr != nil {
				return nil, dialErr
			}
			tlsConn := tls.Client(conn, clientTLSConfig)
			return tlsConn, tlsConn.Handshake()
		},
	})
	if err != nil {
		log.Errorf("Unable to dial borda, will not use gRPC: %v", err)
	} else {
		log.Debug("Using RPC to communicate with borda")
		opts.RPCClient = rc
	}

	bordaClient := borda.NewClient(opts)
	reportToBorda := bordaClient.ReducingSubmitter("proxy_results", maxBufferSize)

	ops.RegisterReporter(func(failure error, ctx map[string]interface{}) {
		if !inSample(ctx) {
			return
		}

		values := map[string]borda.Val{}
		if failure != nil {
			values["error_count"] = borda.Float(1)
		} else {
			values["success_count"] = borda.Float(1)
		}

		reportErr := reportToBorda(values, ctx)
		if reportErr != nil {
			log.Errorf("Error reporting error to borda: %v", reportErr)
		}
	})

	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		if !final {
			// We report only the final values
			return
		}

		if !inSample(ctx) {
			return
		}

		ctx["op"] = "xfer"
		vals := map[string]borda.Val{
			"server_bytes_sent":   borda.Float(stats.SentTotal),
			"server_bps_sent_min": borda.Min(stats.SentMin),
			"server_bps_sent_max": borda.Max(stats.SentMax),
			"server_bps_sent_avg": borda.WeightedAvg(stats.SentAvg, float64(stats.SentTotal)),
			"server_bytes_recv":   borda.Float(stats.RecvTotal),
			"server_bps_recv_min": borda.Min(stats.RecvMin),
			"server_bps_recv_max": borda.Max(stats.RecvMax),
			"server_bps_recv_avg": borda.WeightedAvg(stats.RecvAvg, float64(stats.RecvTotal)),
		}
		reportErr := reportToBorda(vals, ctx)
		if reportErr != nil {
			log.Errorf("Error reporting error to borda: %v", reportErr)
		}
	}
}
