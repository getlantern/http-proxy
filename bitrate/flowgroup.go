package bitrate

import (
	"github.com/fatih/set"
	"github.com/mxk/go-flowrate/flowrate"
)

type FlowGroup struct {
	flows *set.Set
	limit int64
}

func NewFlowGroup(limit int64, ls ...*flowrate.Limiter) *FlowGroup {
	return &FlowGroup{
		flows: set.New(ls),
		limit: limit,
	}
}

func (f *FlowGroup) AddLimiter(l *flowrate.Limiter) {
	f.flows.Add(l)
}

func (f *FlowGroup) RemoveLimiter(l *flowrate.Limiter) (isDone bool) {
	f.flows.Remove(l)
	return f.flows.IsEmpty()
}

func (f *FlowGroup) UpdateLimits() {
	nFlows := f.flows.Size()
	avgLimit := float64(f.limit) / float64(nFlows)
	f.flows.Each(func(item interface{}) bool {
		(*item.(*flowrate.Limiter)).SetLimit(int64(avgLimit))
		return true
	})
}
