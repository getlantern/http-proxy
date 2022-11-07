package testlog

import (
	"os"
	"sync"
	"testing"

	"github.com/getlantern/golog"
)

// Capture captures logs to the given testing.T's Log function.
// Returns a function that stops capturing logs.
//
// Typical usage:
//
//    func MyTest(t *testing.T) {
//        stopCapture := testlog.Capture(t)
//        defer stopCapture()
//        // do stuff
//    }
//
func Capture(t *testing.T) func() {
	w := &testLogWriter{T: t}
	reset := golog.SetOutputs(w, w)
	return func() {
		reset()
		w.stop()
	}
}

type testLogWriter struct {
	*testing.T
	mu      sync.RWMutex
	stopped bool
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.stopped {
		// After writer stopped, just log to console
		p = append([]byte("(logged after test capture stopped) "), p...)
		_, err := os.Stderr.Write(p)
		return len(p), err
	}
	w.Log(string(p))
	return len(p), nil
}

func (w *testLogWriter) stop() {
	w.mu.Lock()
	w.stopped = true
	w.mu.Unlock()
}
