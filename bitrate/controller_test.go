package bitrate

import (
	"io"
	"testing"
	"time"

	"github.com/mxk/go-flowrate/flowrate"
)

type DummyReader struct {
	n int
}

func (r *DummyReader) Read(p []byte) (n int, err error) {
	n = len(p)
	r.n += n
	log.Tracef("Read %v bytes", n)
	return
}

func TestSharedFlowController(t *testing.T) {
	ratelimit := int64(10)

	sc := NewSharedFlowController(&SharedFlowControllerOptions{
		RebalanceInterval: time.Second,
		FlowGroupOpts: &FlowGroupOptions{
			RateLimit:       ratelimit,
			Utilization:     0.9,
			AttenuationStep: 0.1,
			MaxAttenuation:  0.9,
		},
	})

	r, w := io.Pipe()

	limR := flowrate.NewReader(r, ratelimit)

	sc.AddToGroup("mydevice", limR)

}
