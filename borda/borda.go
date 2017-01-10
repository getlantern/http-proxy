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
	reportToBorda := bordaClient.ReducingSubmitter("proxy_results", 10000, func(existingValues map[string]float64, newValues map[string]float64) {
		for key, value := range newValues {
			existingValues[key] += value
		}
	})
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

		values := map[string]float64{}
		if failure != nil {
			values["error_count"] = 1
		} else {
			values["success_count"] = 1
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
		vals := map[string]float64{
			"server_bytes_sent":   float64(stats.SentTotal),
			"server_bps_sent_min": stats.SentMin,
			"server_bps_sent_max": stats.SentMax,
			"server_bps_sent_avg": stats.SentAvg,
			"server_bytes_recv":   float64(stats.RecvTotal),
			"server_bps_recv_min": stats.RecvMin,
			"server_bps_recv_max": stats.RecvMax,
			"server_bps_recv_avg": stats.RecvAvg,
		}
		reportToBorda(vals, ctx)
	}
}
