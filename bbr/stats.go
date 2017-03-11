package bbr

import (
	"math"
	"sync"

	"github.com/getlantern/ema"
	"github.com/gonum/stat"
)

const (
	limit        = 25
	abeThreshold = 2500000
)

type stats struct {
	sent    []float64
	abe     []float64
	weights []float64
	emaABE  *ema.EMA
	size    int
	idx     int
	mx      sync.Mutex
}

func newStats() *stats {
	return &stats{
		sent:    make([]float64, limit),
		abe:     make([]float64, limit),
		weights: make([]float64, limit),
		emaABE:  ema.New(0, 0.5),
	}
}

func (s *stats) update(sent float64, abe float64) {
	if sent < 1024 {
		// Don't bother recording values below 1 KB
		return
	}
	logOfSent := math.Log1p(sent)
	logOfABE := math.Log1p(abe)
	s.mx.Lock()
	s.sent[s.idx] = logOfSent
	s.abe[s.idx] = logOfABE
	s.weights[s.idx] = sent // give more weight to measurements from larger bytes sent
	s.idx++
	if s.idx == limit {
		// Wrap
		s.idx = 0
	}
	if s.size < limit {
		s.size++
	}
	hasDelta := false
	for i := 1; i < s.size; i++ {
		if s.sent[i] != s.sent[i-1] {
			hasDelta = true
			break
		}
	}
	if !hasDelta {
		// There's no way to apply a regression, just ignore
		s.mx.Unlock()
		return
	}

	// Estimate by applying a linear regression
	alpha, beta := stat.LinearRegression(s.sent, s.abe, s.weights, false)
	s.mx.Unlock()
	newEstimate := math.Expm1(alpha + beta*math.Log1p(abeThreshold))
	if math.IsNaN(newEstimate) {
		// ignore
		return
	}
	if newEstimate < 0 {
		newEstimate = 0
	}
	updated := s.emaABE.Update(newEstimate)
	if updated <= 0 {
		// Set estimate to a small value to show that we have something
		s.emaABE.Set(0.01)
	}
}

// estABE estimates the ABE at bytes_sent = 2.5 MB using a logarithmic
// regression on the most recent measurements
func (s *stats) estABE() float64 {
	return s.emaABE.Get()
}
