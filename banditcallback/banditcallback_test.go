package banditcallback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmitter_DisabledWhenUnconfigured(t *testing.T) {
	// Empty token disables emission entirely so non-bandit-eligible
	// tracks can carry the same daemon binary without firing
	// callbacks. Verify Enabled and that EmitIfFirstSeen is a no-op.
	cases := []struct {
		name        string
		token       string
		callbackURL string
	}{
		{"empty token", "", "https://api.example/v1/bandit/callback"},
		{"empty url", "arm-xyz", ""},
		{"both empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := New(tc.token, tc.callbackURL, time.Minute)
			if e.Enabled() {
				t.Fatal("expected disabled")
			}
			e.EmitIfFirstSeen(context.Background(), "did-1", "")
			emitted, _ := e.Stats()
			if emitted != 0 {
				t.Fatalf("expected 0 emits, got %d", emitted)
			}
		})
	}
}

func TestEmitter_FirstSeenFires(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if got := r.URL.Query().Get("token"); got != "arm-test" {
			t.Errorf("token mismatch: %q", got)
		}
		if got := r.URL.Query().Get("did"); got == "" {
			t.Error("missing did")
		}
		if got := r.Header.Get("True-Client-IP"); got != "203.0.113.7" {
			t.Errorf("True-Client-IP mismatch: %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	e := New("arm-test", srv.URL, time.Minute)
	e.EmitIfFirstSeen(context.Background(), "device-a", "203.0.113.7")

	// Emission is async; wait for the goroutine. 1s ceiling is generous.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 callback hit, got %d", got)
	}

	emitted, suppressed := e.Stats()
	if emitted != 1 || suppressed != 0 {
		t.Fatalf("counters: emitted=%d suppressed=%d", emitted, suppressed)
	}
}

func TestEmitter_DedupSuppressesRepeat(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	e := New("arm-test", srv.URL, time.Minute)
	for i := 0; i < 10; i++ {
		e.EmitIfFirstSeen(context.Background(), "device-a", "203.0.113.7")
	}

	// One emit, nine suppressed. Wait for async emit to complete.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		emitted, suppressed := e.Stats()
		if emitted == 1 && suppressed == 9 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	emitted, suppressed := e.Stats()
	if emitted != 1 || suppressed != 9 {
		t.Fatalf("counters: emitted=%d suppressed=%d", emitted, suppressed)
	}
	// Confirm the server only received one hit.
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 callback hit, got %d", got)
	}
}

func TestEmitter_ConcurrentFirstSeenIsSingleFire(t *testing.T) {
	// 100 goroutines racing on the same device-id must yield exactly
	// one outbound call. This is the contention case the mu lock is
	// designed for; a TOCTOU bug here would explode reward signal.
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	e := New("arm-test", srv.URL, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.EmitIfFirstSeen(context.Background(), "race-device", "")
		}()
	}
	wg.Wait()

	// Wait for async send.
	time.Sleep(200 * time.Millisecond)
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected exactly 1 hit under contention, got %d", got)
	}
}

func TestEmitter_ReEmitsAfterTTL(t *testing.T) {
	// After the TTL window, a returning device should re-emit. Use a
	// tiny TTL to keep the test fast.
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	e := New("arm-test", srv.URL, 50*time.Millisecond)
	e.EmitIfFirstSeen(context.Background(), "ttl-device", "")
	time.Sleep(20 * time.Millisecond) // within TTL — suppressed
	e.EmitIfFirstSeen(context.Background(), "ttl-device", "")
	time.Sleep(80 * time.Millisecond) // past TTL — fires again
	e.EmitIfFirstSeen(context.Background(), "ttl-device", "")

	time.Sleep(100 * time.Millisecond)
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 hits across TTL window, got %d", got)
	}
}

func TestEmitter_OmitsTrueClientIPWhenEmpty(t *testing.T) {
	// Empty clientIP must NOT set the header — leaving it absent
	// lets the API fall through to its existing RemoteAddr-based
	// MaxMind lookup. Setting an empty header would shadow that
	// fallback with a junk value.
	var sawHeader int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["True-Client-Ip"]; ok {
			atomic.StoreInt32(&sawHeader, 1)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	e := New("arm-test", srv.URL, time.Minute)
	e.EmitIfFirstSeen(context.Background(), "device-x", "")
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&sawHeader) != 0 {
		t.Fatal("True-Client-IP must be absent when clientIP is empty")
	}
}
