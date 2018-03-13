package api

import (
	"sync"
	"time"

	"github.com/chromium/hstspreload.org/database"
)

type statesEntry struct {
	domains   []database.DomainState
	cacheTime time.Time
}

type stateEntry struct {
	state     database.DomainState
	cacheTime time.Time
}

type cache struct {
	lock            sync.Mutex
	domainsByStatus map[database.PreloadStatus]statesEntry
	stateForDomain  map[string]stateEntry
	cacheDuration   time.Duration
}

func cacheWithDuration(duration time.Duration) *cache {
	return &cache{
		domainsByStatus: make(map[database.PreloadStatus]statesEntry),
		stateForDomain:  make(map[string]stateEntry),
		cacheDuration:   duration,
	}
}

func (api API) statesWithStatusCached(status database.PreloadStatus) ([]database.DomainState, error) {
	api.cache.lock.Lock()
	defer api.cache.lock.Unlock()

	if entry, ok := api.cache.domainsByStatus[status]; ok {
		if time.Since(entry.cacheTime) < api.cache.cacheDuration {
			return entry.domains, nil
		}
	}

	domains, err := api.database.StatesWithStatus(status)
	if err != nil {
		return domains, err
	}

	api.cache.domainsByStatus[status] = statesEntry{
		domains:   domains,
		cacheTime: time.Now(),
	}

	return domains, nil
}

func (api API) stateForDomainCached(domain string) (state database.DomainState, err error) {
	api.cache.lock.Lock()
	defer api.cache.lock.Unlock()

	if entry, ok := api.cache.stateForDomain[domain]; ok {
		if time.Since(entry.cacheTime) < api.cache.cacheDuration {
			return entry.state, nil
		}
	}

	state, err = api.database.StateForDomain(domain)
	if err != nil {
		return state, err
	}

	api.cache.stateForDomain[domain] = stateEntry{
		state:     state,
		cacheTime: time.Now(),
	}

	return state, nil
}
