package borda

import (
	"encoding/base64"
	"encoding/binary"
	"time"

	borda "github.com/getlantern/borda/client"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
	"github.com/getlantern/ops"
)

var (
	log = golog.LoggerFor("lantern-proxy-borda")
)

// Enable enables borda reporting
func Enable(bordaReportInterval time.Duration, bordaSamplePercentage float64) listeners.MeasuredReportFN {
	bordaClient := borda.NewClient(&borda.Options{
		BatchInterval: bordaReportInterval,
	})
	reportToBorda := bordaClient.ReducingSubmitter("proxy_results", 10000)
	ops.RegisterReporter(func(failure error, ctx map[string]interface{}) {
		// Sample a subset of device ids
		deviceID := ctx["device_id"]
		if deviceID == nil {
			log.Debugf("No device_id, not reporting measurement")
			return
		}

		// DeviceID is expected to be a Base64 encoded 48-bit (6 byte) MAC address
		deviceIDBytes, base64Err := base64.StdEncoding.DecodeString(deviceID.(string))
		if base64Err != nil {
			log.Debugf("Error decoding base64 deviceID: %v", base64Err)
			return
		}
		var deviceIDInt uint64
		if len(deviceIDBytes) != 6 {
			log.Debugf("Unexpected DeviceID length %d", len(deviceIDBytes))
			return
		}
		// Pad and decode to int
		paddedDeviceIDBytes := append(deviceIDBytes, 0, 0)
		// Use BigEndian because Mac address has most significant bytes on left
		deviceIDInt = binary.BigEndian.Uint64(paddedDeviceIDBytes)
		if deviceIDInt%uint64(1/bordaSamplePercentage) != 0 {
			log.Trace("DeviceID not being sampled")
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
