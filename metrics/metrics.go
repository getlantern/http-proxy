package metrics

import (
	"fmt"
	"time"

	"github.com/rcrowley/go-metrics"
)

type MovingAverage interface {
	Update(n int64)

	Rate1Min() float64

	Rate5Min() float64

	Rate15Min() float64
}

type movingAverage struct {
	min1  metrics.EWMA
	min5  metrics.EWMA
	min15 metrics.EWMA
}

func NewMovingAverage() MovingAverage {
	ma := &movingAverage{
		min1:  metrics.NewEWMA1(),
		min5:  metrics.NewEWMA5(),
		min15: metrics.NewEWMA15(),
	}
	go func() {
		for {
			// Advance the time on the moving averages. This is supposed to happen
			// every 5 seconds.
			time.Sleep(5 * time.Second)
			ma.min1.Tick()
			ma.min5.Tick()
			ma.min15.Tick()
		}
	}()
	return ma
}

func (ma *movingAverage) Update(n int64) {
	ma.min1.Update(n)
	ma.min5.Update(n)
	ma.min15.Update(n)
}

func (ma *movingAverage) Rate1Min() float64 {
	return ma.min1.Rate()
}

func (ma *movingAverage) Rate5Min() float64 {
	return ma.min5.Rate()
}

func (ma *movingAverage) Rate15Min() float64 {
	return ma.min15.Rate()
}

func (ma *movingAverage) String() string {
	return fmt.Sprintf("%10.f %10.f %10.f", ma.Rate1Min(), ma.Rate5Min(), ma.Rate15Min())
}
