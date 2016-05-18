// SharedFlowManager controls the individual stream bitrate limits
// associated to a group of streams

package bitrate

import (
	"errors"
	"sync"
)

var (
	errorGroupNotExist = errors.New("Group does not exist")
)

// TODO
type Status int

type Limiter interface {
	Done() int64
	Status() Status
	SetTransferSize(bytes int64)
	SetLimit(new int64) (old int64)
	SetBlocking(new bool) (old bool)
}

type SharedFlowControllerOptions struct {
	globalLimit int64
}

type GroupsMap map[string]*FlowGroup

type SharedFlowController struct {
	options *SharedFlowControllerOptions
	gMtx    sync.RWMutex
	groups  GroupsMap
}

func NewSharedFlowController(opts *SharedFlowControllerOptions) *SharedFlowController {
	s := &SharedFlowController{
		options: opts,
		groups:  make(GroupsMap),
	}
	go s.updateProc()

	return s
}

func (m *SharedFlowController) AddToGroup(group string, l *Limiter) (isNew bool) {
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

func (m *SharedFlowController) RemoveFromGroup(flowgroup string, l *Limiter) error {
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

func (m *SharedFlowController) updateProc() {
	m.gMtx.Lock()
	defer m.gMtx.Unlock()

	for _, fg := range m.groups {
		fg.UpdateLimits()
	}
}
