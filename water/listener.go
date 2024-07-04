package water

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v0"
)

var log = golog.LoggerFor("water")

// NewWATERListener creates a WATER listener
// Currently water doesn't support customized TCP connections and we need to listen and receive requests directly from the WATER listener
func NewWATERListener(ctx context.Context, transport string, baseListener net.Listener, wasm string) (net.Listener, error) {
	decodedWASM, err := base64.StdEncoding.DecodeString(wasm)
	if err != nil {
		log.Errorf("failed to decode WASM base64: %v", err)
		return nil, err
	}

	cfg := &water.Config{
		TransportModuleBin: decodedWASM,
		NetworkListener:    baseListener,
		OverrideLogger:     slog.New(newLogHandler(log, transport)),
	}

	waterListener, err := water.NewListenerWithContext(ctx, cfg)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	return waterListener, nil
}
