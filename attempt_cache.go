package cascadht

import (
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	"github.com/libp2p/go-libp2p/core/peer"
)

type (
	peerRoutingAttemptCache struct {
		lock     sync.Mutex
		lru      *lru.Cache
		maxSince time.Duration
	}
	attempt struct {
		at time.Time
	}
)

func newPeerRoutingAttemptCache(maxEntries int, maxAge time.Duration) *peerRoutingAttemptCache {
	return &peerRoutingAttemptCache{
		lru:      lru.New(maxEntries),
		maxSince: maxAge,
	}
}

func (p *peerRoutingAttemptCache) attempt(id peer.ID) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	if v, found := p.lru.Get(id); found {
		if latest, ok := v.(*attempt); ok && latest != nil {
			if time.Since(latest.at) < p.maxSince {
				return false
			}
		}
	}
	p.lru.Add(id, &attempt{at: time.Now()})
	return true
}
