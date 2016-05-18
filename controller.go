// SharedFlowManager controls the individual stream bitrate limits
// associated to a group of streams

package bitrate

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/mxk/go-flowrate/flowrate"
)

var (
	errorGroupNotExist = errors.New("Group does not exist")
)

type SharedFlowControllerOptions struct {
	GlobalLimit int64
	Interval    time.Duration
}

type GroupsMap map[string]*FlowGroup

type SharedFlowController struct {
	options *SharedFlowControllerOptions
	ticker  *time.Ticker
	gMtx    sync.RWMutex
	groups  GroupsMap
}

func NewSharedFlowController(opts *SharedFlowControllerOptions) *SharedFlowController {
	if opts.GlobalLimit == 0 {
		opts.GlobalLimit = math.MaxInt64
	}
	if opts.Interval == 0 {
		opts.Interval = time.Second
	}

	s := &SharedFlowController{
		options: opts,
		groups:  make(GroupsMap),
		ticker:  time.NewTicker(opts.Interval),
	}
	go s.updateFlowGroups()

	return s
}

func (m *SharedFlowController) AddToGroup(group string, l *flowrate.Limiter) (isNew bool) {
	m.gMtx.Lock()
	defer m.gMtx.Unlock()
	if s, ok := m.groups[group]; ok {
		s.AddLimiter(l)
		isNew = false
	} else {
		//newSet := set.New(l)
		// TOD
		flowgroup := NewFlowGroup(9999999, l)
		m.groups[group] = flowgroup
		isNew = true
	}
	return
}

func (m *SharedFlowController) RemoveFromGroup(flowgroup string, l *flowrate.Limiter) error {
	m.gMtx.Lock()
	defer m.gMtx.Unlock()
	fg, ok := m.groups[flowgroup]
	if !ok {
		return errorGroupNotExist
	}

	if fg.RemoveLimiter(l) {
		delete(m.groups, flowgroup)
	}
	return nil
}

func (m *SharedFlowController) updateFlowGroups() {
	m.gMtx.Lock()
	defer m.gMtx.Unlock()

	for range m.ticker.C {
		for _, fg := range m.groups {
			fg.UpdateLimits()
		}
	}
}
