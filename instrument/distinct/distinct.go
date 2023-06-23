package distinct

import (
	"hash"
	"hash/fnv"
	"sync"
	"time"
)

// A SlidingWindowDistinctCount is a utility that tracks the exact cardinality
// of a set of strings over a sliding time window. In comparison to
// sophisticated algorithms like HyperLogLog, it uses considerably more memory.
type SlidingWindowDistinctCount struct {
	bucketSize time.Duration

	buckets []map[uint32]bool
	t       time.Time
	h       hash.Hash32
	idx     int
	sync.RWMutex
}

func NewSlidingWindowDistinctCount(windowSize, bucketSize time.Duration) *SlidingWindowDistinctCount {
	buckets := make([]map[uint32]bool, windowSize/bucketSize)
	for i := range buckets {
		buckets[i] = make(map[uint32]bool)
	}

	return &SlidingWindowDistinctCount{
		bucketSize: bucketSize,
		buckets:    buckets,
		t:          time.Now().Truncate(bucketSize),
		h:          fnv.New32(),
	}
}

func (w *SlidingWindowDistinctCount) Add(v string) {
	w.Lock()
	defer w.Unlock()

	t := time.Now().Truncate(w.bucketSize)
	elapsed := t.Sub(w.t)
	for i := 1; i <= int(elapsed/w.bucketSize); i++ {
		// Move forward in the ring buffer and clear the next bucket.
		w.idx = (w.idx + 1) % len(w.buckets)
		w.buckets[w.idx] = make(map[uint32]bool, len(w.buckets[w.idx]))
	}

	w.h.Reset()
	w.h.Write([]byte(v))
	w.buckets[w.idx][w.h.Sum32()] = true
	w.t = t
}

func (w *SlidingWindowDistinctCount) Cardinality() int {
	w.RLock()
	defer w.RUnlock()

	set := make(map[uint32]bool, len(w.buckets[0]))
	for _, b := range w.buckets {
		for v := range b {
			set[v] = true
		}
	}

	return len(set)
}
