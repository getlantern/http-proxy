package distinct

import (
	"math/rand"
	"testing"
	"time"
)

func BenchmarkSlidingWindowDistinctCount(b *testing.B) {
	b.ReportAllocs()

	vals := make([]string, 0, 1000)
	set := make(map[string]bool)
	const chars = "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < cap(vals); i++ {
		b := make([]byte, 10)
		for i := range b {
			b[i] = chars[rand.Intn(len(chars))]
		}

		vals = append(vals, string(b))
		set[string(b)] = true
	}

	configurations := []struct {
		Name                   string
		windowSize, bucketSize time.Duration
	}{
		{
			Name:       "compact",
			bucketSize: time.Microsecond,
			windowSize: time.Millisecond,
		},
		{
			Name:       "wide",
			bucketSize: time.Microsecond,
			windowSize: time.Second,
		},
		{
			Name:       "tall",
			bucketSize: 100 * time.Millisecond,
			windowSize: 200 * time.Millisecond,
		},
	}

	for _, conf := range configurations {
		b.Run(conf.Name, func(b *testing.B) {
			w := NewSlidingWindowDistinctCount(conf.windowSize, conf.bucketSize)
			var c int

			b.Run("write", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					w.Add(vals[c%len(vals)])
					c = (c + 1%len(vals))
				}
			})

			b.Run("read", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					w.Cardinality()
				}
			})

			var total, min, max int
			min = len(w.buckets[0])
			for _, bucket := range w.buckets {
				total += len(bucket)
				if len(bucket) > max {
					max = len(bucket)
				} else if len(bucket) < min {
					min = len(bucket)
				}
			}

			b.Logf("buckets: %d", len(w.buckets))
			b.Logf("total values: %d", total)
			b.Logf("min height: %d", min)
			b.Logf("max height: %d", max)
			b.Logf("average height: %0.2f", float64(total)/float64(len(w.buckets)))

			cardinality := w.Cardinality()
			if cardinality != len(set) {
				b.Logf("unexpected cardinality: %d", cardinality)
				b.Fail()
			}
		})
	}

}
