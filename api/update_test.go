package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

const ()

// Testing that PolicyType is populated within the database when the Update Endpoint is called
func TestPolicyType(t *testing.T) {
	api, mc, h, c := mockAPI(0 * time.Second)

	api.bulkPreloaded["removal-preloaded-bulk-eligible.test"] = true
	api.bulkPreloaded["removal-not-preloaded-bulk-eligible.test"] = true
	api.bulkPreloaded["removal-preloaded-bulk-ineligible.test"] = true

	pr1 := map[string]hstspreload.Issues{
		"garron.net":                      emptyIssues,
		"badssl.com":                      issuesWithWarnings,
		"example.com":                     issuesWithErrors,
		"removal-pending-eligible.test":   emptyIssues,
		"removal-pending-ineligible.test": emptyIssues,
	}
	rr1 := map[string]hstspreload.Issues{
		"removal-preloaded-bulk-eligible.test":     emptyIssues,
		"removal-preloaded-not-bulk-eligible.test": emptyIssues,
		"removal-preloaded-bulk-ineligible.test":   issuesWithErrors,
		"removal-pending-eligible.test":            emptyIssues,
		"removal-pending-ineligible.test":          issuesWithErrors,
	}

	// Policy types set to preloadlist.Test for testing simplicity
	pl1 := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "garron.net", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Test},
		{Name: "chromium.org", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false, Policy: preloadlist.Test},
		{Name: "removal-preloaded-bulk-eligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Test},
		{Name: "removal-preloaded-not-bulk-eligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Test},
		{Name: "removal-preloaded-bulk-ineligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Test},
		{Name: "godoc.og", Mode: "", IncludeSubDomains: true, Policy: preloadlist.Test},
		{Name: "dev", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Test},
	}}

	data1 := MockData{pr1, rr1, pl1}

	jsonContentType := "application/json; charset=utf-8"

	updateTest := apiTestCase{"garron.net pending", data1, failNone, api.Status, "GET", "?domain=garron.net",
		200, jsonContentType, wantBody{state: &database.DomainState{
			Name: "garron.net", Status: database.StatusPending}}}

	h.preloadableResponses = pr1
	h.removableResponses = rr1
	c.list = pl1

	mc.FailCalls = (failNone & failDatabase) != 0
	c.failCalls = (failNone & failChromiumpreload) != 0

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest(updateTest.method, updateTest.url, nil)
	if err != nil {
		t.Fatalf("[%s] %s", updateTest.description, err)
	}

	api.Update(w, r)

	states, err := api.database.AllDomainStates()
	if err != nil {
		t.Fatalf("Couldn't get the states of all domains in the database.")
	}
	for _, state := range states {
		// change to preloadlist.UnspecifiedPolicyType in due time
		if state.Policy != preloadlist.Test {
			t.Errorf("Policy field not accurately populated in the database for %s.", state.Name)
		}
	}
}
