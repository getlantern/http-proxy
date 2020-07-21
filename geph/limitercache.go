package geph

import (
	"math/rand"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/time/rate"
)

type limitFactory struct {
	lcache *lru.Cache
	limit  rate.Limit
}

func (lf *limitFactory) getLimiter(sessid [32]byte) *rate.Limiter {
	limit := rate.Limit(float64(lf.limit) * (0.9 + rand.Float64()/5))
	new := rate.NewLimiter(limit, 1024*1024)
	prev, _, _ := lf.lcache.PeekOrAdd(sessid, new)
	if prev != nil {
		return prev.(*rate.Limiter)
	}
	log.Debugf("limit for sessid is %v", limit)
	return new
}

func newLimitFactory(limit rate.Limit) *limitFactory {
	l, _ := lru.New(65536)
	return &limitFactory{
		limit:  limit,
		lcache: l,
	}
}

var slowLimitFactory = newLimitFactory(100 * 1024)
