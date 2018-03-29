package api

import (
	"testing"
	"time"

	"github.com/chromium/hstspreload.org/database"
)

func TestCacheZeroDuration(t *testing.T) {
	api, mc, _, _ := mockAPI(0 * time.Second)

	domainA := database.DomainState{Name: "a.test", Status: database.StatusPending}
	domainB := database.DomainState{Name: "b.test", Status: database.StatusPending}
	domainC := database.DomainState{Name: "c.test", Status: database.StatusPendingRemoval}

	api.database.PutState(domainA)

	domains, err := api.statesWithStatusCached(database.StatusPending)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("First pending retrieval had wrong number of domains: %d", len(domains))
	}
	if domains[0] != domainA {
		t.Fatalf("First pending retrieval had wrong domain: %v", domains[0])
	}

	api.database.PutState(domainB)
	api.database.PutState(domainC)

	domains, err = api.statesWithStatusCached(database.StatusPending)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("Second pending retrieval had wrong number of domains: %d", len(domains))
	}

	domains, err = api.statesWithStatusCached(database.StatusPendingRemoval)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("First pending removal retrieval had wrong number of domains: %d", len(domains))
	}
	if domains[0] != domainC {
		t.Fatalf("First pending removal retrieval had wrong domain: %v", domains[0])
	}

	mc.FailCalls = true
	_, err = api.statesWithStatusCached(database.StatusPending)
	if err == nil {
		t.Fatalf("Expected uncached call to fail")
	}
}

func TestCacheShortDuration(t *testing.T) {
	duration := 1 * time.Second
	api, mc, _, _ := mockAPI(duration)

	domainA := database.DomainState{Name: "a.test", Status: database.StatusPending}
	domainB := database.DomainState{Name: "b.test", Status: database.StatusPending}
	domainC := database.DomainState{Name: "c.test", Status: database.StatusPendingRemoval}

	api.database.PutState(domainA)

	domains, err := api.statesWithStatusCached(database.StatusPending)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("First pending retrieval had wrong number of domains: %d", len(domains))
	}
	if domains[0] != domainA {
		t.Fatalf("First pending retrieval had wrong domain: %v", domains[0])
	}

	domains, err = api.statesWithStatusCached(database.StatusPendingRemoval)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("First pending removal retrieval had wrong number of domains: %d", len(domains))
	}

	api.database.PutState(domainB)
	api.database.PutState(domainC)

	domains, err = api.statesWithStatusCached(database.StatusPending)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("Cached pending retrieval had wrong number of domains: %d", len(domains))
	}

	domains, err = api.statesWithStatusCached(database.StatusPendingRemoval)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("Cached pending removal retrieval had wrong number of domains: %d", len(domains))
	}

	mc.FailCalls = true

	domains, err = api.statesWithStatusCached(database.StatusPending)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("Failing database pending retrieval had wrong number of domains: %d", len(domains))
	}

	domains, err = api.statesWithStatusCached(database.StatusPendingRemoval)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("Failing database pending removal retrieval had wrong number of domains: %d", len(domains))
	}

	time.Sleep(duration)
	mc.FailCalls = false

	domains, err = api.statesWithStatusCached(database.StatusPending)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("Last pending retrieval had wrong number of domains: %d", len(domains))
	}

	domains, err = api.statesWithStatusCached(database.StatusPendingRemoval)
	if err != nil {
		t.Fatalf("Error getting domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("Last removal retrieval had wrong number of domains: %d", len(domains))
	}
	if domains[0] != domainC {
		t.Fatalf("Last removal retrieval had wrong domain: %v", domains[0])
	}
}
