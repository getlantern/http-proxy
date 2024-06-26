package water

import (
	"context"
	"encoding/base64"
	"net"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v0"
)

var log = golog.LoggerFor("water")

func NewWATERListener(ctx context.Context, address string, wasm string) (net.Listener, error) {
	decodedWASM, err := base64.StdEncoding.DecodeString(wasm)
	if err != nil {
		log.Errorf("failed to decode WASM base64: %v", err)
		return nil, err
	}

	cfg := &water.Config{
		TransportModuleBin: decodedWASM,
	}

	waterListener, err := cfg.ListenContext(ctx, "tcp", address)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	return waterListener, nil
}
