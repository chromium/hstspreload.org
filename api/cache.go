package api

import (
	"sync"
	"time"

	"github.com/chromium/hstspreload.org/database"
)

type entry struct {
	domains   []database.DomainState
	cacheTime time.Time
}

type cache struct {
	lock            sync.Mutex
	domainsByStatus map[database.PreloadStatus]entry
	cacheDuration   time.Duration
}

func cacheWithDuration(duration time.Duration) *cache {
	return &cache{
		domainsByStatus: make(map[database.PreloadStatus]entry),
		cacheDuration:   duration,
	}
}

func (api API) domainsWithStatusCached(status database.PreloadStatus) ([]database.DomainState, error) {
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

	api.cache.domainsByStatus[status] = entry{
		domains:   domains,
		cacheTime: time.Now(),
	}

	return domains, nil
}
