package bitrate

import "github.com/fatih/set"

type FlowGroup struct {
	flows *set.Set
	limit int64
}

func NewFlowGroup(limit int64, ls ...*Limiter) *FlowGroup {
	return &FlowGroup{
		flows: set.New(ls),
		limit: limit,
	}
}

func (f *FlowGroup) SetLimit(l int64) {
	f.limit = l
}

func (f *FlowGroup) AddLimiter(l *Limiter) {
	f.flows.Add(l)
}

func (f *FlowGroup) RemoveLimiter(l *Limiter) (isDone bool) {
	f.flows.Remove(l)
	return f.flows.IsEmpty()
}

func (f *FlowGroup) UpdateLimits() {
}
