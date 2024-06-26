package water

import (
	"context"
	"encoding/base64"
	"net"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	v1 "github.com/refraction-networking/water/transport/v1"
)

var log = golog.LoggerFor("water")

func NewWATERListener(ctx context.Context, baseListener net.Listener, wasm string) (net.Listener, error) {
	decodedWASM, err := base64.StdEncoding.DecodeString(wasm)
	if err != nil {
		log.Errorf("failed to decode WASM base64: %v", err)
		return nil, err
	}

	cfg := &water.Config{
		TransportModuleBin:  decodedWASM,
		NetworkListener:     baseListener,
		ModuleConfigFactory: water.NewWazeroModuleConfigFactory(),
	}

	waterListener, err := v1.NewListenerWithContext(ctx, cfg)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	return waterListener, nil
}
