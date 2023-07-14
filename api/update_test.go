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

// TestPolicyType tests that PolicyType is populated within the database when the Update endpoint is called.
func TestPolicyType(t *testing.T) {
	api, _, mockHstspreload, mockPreloadlist := mockAPI(0 * time.Second)

	TestPreloadableResponses := map[string]hstspreload.Issues{
		"garron.net":                      emptyIssues,
		"badssl.com":                      issuesWithWarnings,
		"example.com":                     issuesWithErrors,
		"removal-pending-eligible.test":   emptyIssues,
		"removal-pending-ineligible.test": emptyIssues,
	}
	TestRemovableResponses := map[string]hstspreload.Issues{
		"removal-preloaded-bulk-eligible.test":     emptyIssues,
		"removal-preloaded-not-bulk-eligible.test": emptyIssues,
		"removal-preloaded-bulk-ineligible.test":   issuesWithErrors,
		"removal-pending-eligible.test":            emptyIssues,
		"removal-pending-ineligible.test":          issuesWithErrors,
	}

	TestPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "garron.net", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "chromium.org", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false, Policy: preloadlist.Custom},
		{Name: "removal-preloaded-bulk-eligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.UnspecifiedPolicyType},
		{Name: "removal-preloaded-not-bulk-eligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
		{Name: "removal-preloaded-bulk-ineligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Test},
		{Name: "godoc.og", Mode: "", IncludeSubDomains: true, Policy: preloadlist.Custom},
		{Name: "dev", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.UnspecifiedPolicyType},
	}}

	mockHstspreload.preloadableResponses = TestPreloadableResponses
	mockHstspreload.removableResponses = TestRemovableResponses
	mockPreloadlist.list = TestPreloadlist

	// tests for correct behavior when a domain changes from pending status to preloaded status
	testPendingToPreloaded := database.DomainState{Name: "garron.net", Status: database.StatusPending, IncludeSubDomains: true, Policy: preloadlist.Custom}
	api.database.PutState(testPendingToPreloaded)

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}

	api.Update(w, r)

	if w.Code != 200 {
		t.Errorf("HTTP Response Invalid: Status code is not 200")
	}

	states, err := api.database.AllDomainStates()
	if err != nil {
		t.Fatalf("Couldn't get the states of all domains in the database.")
	}

	expectedPolicies := map[string]preloadlist.PolicyType{
		"garron.net":                               preloadlist.Bulk18Weeks,
		"chromium.org":                             preloadlist.Custom,
		"removal-preloaded-bulk-eligible.test":     preloadlist.UnspecifiedPolicyType,
		"removal-preloaded-not-bulk-eligible.test": preloadlist.Bulk1Year,
		"removal-preloaded-bulk-ineligible.test":   preloadlist.Test,
		"godoc.og":                                 preloadlist.Custom,
		"dev":                                      preloadlist.UnspecifiedPolicyType,
	}

	for _, state := range states {
		if state.Policy != expectedPolicies[state.Name] {
			t.Errorf("Policy field not accurately populated in the database for %s with %s", state.Policy, expectedPolicies[state.Name])
		}
	}
}
