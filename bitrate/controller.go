// SharedFlowManager controls the individual stream bitrate limits
// associated to a group of streams

package bitrate

import (
	"errors"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"github.com/mxk/go-flowrate/flowrate"
)

var (
	errorGroupNotExist = errors.New("Group does not exist")
	log                = golog.LoggerFor("bitrate")
)

type SharedFlowControllerOptions struct {
	RebalanceInterval time.Duration
	FlowGroupOpts     *FlowGroupOptions
}

type GroupsMap map[string]*FlowGroup

type SharedFlowController struct {
	options *SharedFlowControllerOptions
	ticker  *time.Ticker
	gMtx    sync.RWMutex
	groups  GroupsMap
}

func NewSharedFlowController(opts *SharedFlowControllerOptions) *SharedFlowController {
	if opts.RebalanceInterval == 0 {
		opts.RebalanceInterval = time.Second
	}
	if opts.FlowGroupOpts == nil {
		panic("FlowGroupOpts should be provided")
	}

	s := &SharedFlowController{
		options: opts,
		groups:  make(GroupsMap),
		ticker:  time.NewTicker(opts.RebalanceInterval),
	}
	go s.updateFlowGroups()

	return s
}

func (m *SharedFlowController) AddToGroup(group string, l *flowrate.Limiter) (isNew bool) {
	m.gMtx.Lock()
	defer m.gMtx.Unlock()
	if fg, ok := m.groups[group]; ok {
		fg.AddLimiter(l)
		isNew = false
		fg.RebalanceLimits()
	} else {
		flowgroup := NewFlowGroup(m.options.FlowGroupOpts, l)
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
			fg.RebalanceLimits()
		}
	}
}
