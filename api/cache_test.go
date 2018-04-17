package api

import (
	"testing"
	"time"

	"github.com/chromium/hstspreload.org/database"
)

func TestCacheZeroDuration(t *testing.T) {
	api, mc, _, _ := mockAPI(0 * time.Second)

	domainA := database.DomainState{Name: "a.test", Status: database.StatusPending}
	domainB := database.DomainState{Name: "b.test", Status: database.StatusPending, IncludeSubDomains: true}
	domainC := database.DomainState{Name: "c.test", Status: database.StatusPendingRemoval}
	newDomainA := database.DomainState{Name: "a.test", Status: database.StatusPending, IncludeSubDomains: false}

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

	state, err := api.stateForDomainCached("a.test")
	if err != nil {
		t.Fatalf("Error getting state for domain a.test: %v", err)
	}
	if state != domainA {
		t.Fatalf("State of a.test is incorrect: %v", state)
	}

	api.database.PutState(domainB)
	api.database.PutState(domainC)
	api.database.PutState(newDomainA)

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

	state, err = api.stateForDomainCached("a.test")
	if err != nil {
		t.Fatalf("Error getting state for domain a.test: %v", err)
	}
	if state != newDomainA {
		t.Fatalf("State of a.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("b.test")
	if err != nil {
		t.Fatalf("Error getting state for domain b.test: %v", err)
	}
	if state != domainB {
		t.Fatalf("State of b.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("c.test")
	if err != nil {
		t.Fatalf("Error getting state for domain c.test: %v", err)
	}
	if state != domainC {
		t.Fatalf("State of c.test is incorrect: %v", state)
	}

	mc.FailCalls = true
	_, err = api.statesWithStatusCached(database.StatusPending)
	if err == nil {
		t.Fatalf("Expected uncached call StatesWithStatus to fail")
	}
	_, err = api.stateForDomainCached("")
	if err == nil {
		t.Fatalf("Expected uncached call StateForDomain to fail")
	}
}

func TestCacheShortDuration(t *testing.T) {
	duration := 1 * time.Second
	api, mc, _, _ := mockAPI(duration)

	domainA := database.DomainState{Name: "a.test", Status: database.StatusPending}
	domainB := database.DomainState{Name: "b.test", Status: database.StatusPending, IncludeSubDomains: true}
	domainC := database.DomainState{Name: "c.test", Status: database.StatusPendingRemoval}
	newDomainA := database.DomainState{Name: "a.test", Status: database.StatusPending, IncludeSubDomains: true}

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

	state, err := api.stateForDomainCached("a.test")
	if err != nil {
		t.Fatalf("Error getting state for domain a.test: %v", err)
	}
	if state != domainA {
		t.Fatalf("State of a.test is incorrect: %v", state)
	}

	api.database.PutState(domainB)
	api.database.PutState(domainC)
	api.database.PutState(newDomainA)

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

	state, err = api.stateForDomainCached("a.test")
	if err != nil {
		t.Fatalf("Error getting state for domain a.test: %v", err)
	}
	if state != domainA {
		t.Fatalf("Cached state retrieval of a.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("b.test")
	if err != nil {
		t.Fatalf("Error getting state for domain b.test: %v", err)
	}
	if state != domainB {
		t.Fatalf("Cached state retrieval of b.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("c.test")
	if err != nil {
		t.Fatalf("Error getting state for domain c.test: %v", err)
	}
	if state != domainC {
		t.Fatalf("Cached state retrival of c.test is incorrect: %v", state)
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

	state, err = api.stateForDomainCached("a.test")
	if err != nil {
		t.Fatalf("Error getting state for domain a.test: %v", err)
	}
	if state != domainA {
		t.Fatalf("Failing state retrieval of a.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("b.test")
	if err != nil {
		t.Fatalf("Error getting state for domain b.test: %v", err)
	}
	if state != domainB {
		t.Fatalf("Failing state retrival of b.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("c.test")
	if err != nil {
		t.Fatalf("Error getting state for domain c.test: %v", err)
	}
	if state != domainC {
		t.Fatalf("Failing state retrival of c.test is incorrect: %v", state)
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

	state, err = api.stateForDomainCached("a.test")
	if err != nil {
		t.Fatalf("Error getting state for domain a.test: %v", err)
	}
	if state != newDomainA {
		t.Fatalf("Last state retrieval of a.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("b.test")
	if err != nil {
		t.Fatalf("Error getting state for domain b.test: %v", err)
	}
	if state != domainB {
		t.Fatalf("Last state retrival of b.test is incorrect: %v", state)
	}
	state, err = api.stateForDomainCached("c.test")
	if err != nil {
		t.Fatalf("Error getting state for domain c.test: %v", err)
	}
	if state != domainC {
		t.Fatalf("Last state retrival of c.test is incorrect: %v", state)
	}
}
