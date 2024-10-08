package water

import (
	"context"
	"log/slog"
	"net"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v1"
)

var log = golog.LoggerFor("water")

// NewWATERListener creates a WATER listener
// Currently water doesn't support customized TCP connections and we need to listen and receive requests directly from the WATER listener
func NewWATERListener(ctx context.Context, baseListener net.Listener, transport, address string, wasm []byte) (net.Listener, error) {
	cfg := &water.Config{
		TransportModuleBin: wasm,
		OverrideLogger:     slog.New(newLogHandler(log, transport)),
	}

	if baseListener != nil {
		cfg.NetworkListener = baseListener
	}

	waterListener, err := cfg.ListenContext(ctx, "tcp", address)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	return waterListener, nil
}
