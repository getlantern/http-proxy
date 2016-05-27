package borda

import (
	"encoding/base64"
	"encoding/binary"
	"time"

	borda "github.com/getlantern/borda/client"
	"github.com/getlantern/golog"
	"github.com/getlantern/ops"
)

var (
	log = golog.LoggerFor("lantern-proxy-borda")
)

// Enable enables borda reporting
func Enable(bordaReportInterval time.Duration, bordaSamplePercentage float64) {
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
			log.Debugf("No device id, not reporting measurement")
			return
		}
		deviceIDBytes, base64Err := base64.StdEncoding.DecodeString(deviceID.(string))
		if base64Err != nil {
			log.Debugf("Error decoding base64 deviceID: %v", base64Err)
			return
		}
		var deviceIDInt uint64
		if len(deviceIDBytes) < 4 {
			log.Debugf("DeviceID too small: %v", base64Err)
		} else if len(deviceIDBytes) < 8 {
			deviceIDInt = uint64(binary.BigEndian.Uint32(deviceIDBytes))
		} else {
			deviceIDInt = binary.BigEndian.Uint64(deviceIDBytes)
		}
		if deviceIDInt%uint64(1/bordaSamplePercentage) != 0 {
			log.Trace("DeviceID not being sampled")
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
}
