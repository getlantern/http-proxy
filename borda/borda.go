package borda

import (
	"math/rand"
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
)

// Enable enables borda reporting
func Enable(bordaReportInterval time.Duration, bordaSamplePercentage float64) listeners.MeasuredReportFN {
	inSample := func() bool {
		return rand.Float64() < bordaSamplePercentage
	}

	opts := &borda.Options{
		BatchInterval: bordaReportInterval,
	}

	rc, err := rpc.Dial("borda.getlantern.org:17712", &rpc.ClientOpts{})
	if err != nil {
		log.Errorf("Unable to dial borda, will not use gRPC: %v", err)
	} else {
		log.Debug("Using RPC to communicate with borda")
		opts.RPCClient = rc
	}

	bordaClient := borda.NewClient(opts)
	reportToBorda := bordaClient.ReducingSubmitter("proxy_results", 10000)

	ops.RegisterReporter(func(failure error, ctx map[string]interface{}) {
		if !inSample() {
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

		if !inSample() {
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
		reportToBorda(vals, ctx)
	}
}
