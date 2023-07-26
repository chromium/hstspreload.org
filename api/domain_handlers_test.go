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

var normalizeDomainTests = []struct {
	input    string
	expected string
}{
	{"example.com", "example.com"},
	{"EXAMPLE.com", "example.com"},
	{"example.рф", "example.xn--p1ai"},
	{"WWW.müller.de", "www.xn--mller-kva.de"},
	{"eXamPle.coM", "example.com"},
}

func TestNormalizeDomain(t *testing.T) {
	for _, c := range normalizeDomainTests {
		result, err := normalizeDomain(c.input)
		if err != nil {
			t.Error(err)
		}
		if c.expected != result {
			t.Errorf("normalizeDomain(%q) => %q, want %q", c.input, result, c.expected)
		}
	}
}

// TestIneligible tests that IneligibleDomainState Database is populated when the Ineligible endpoint is called.
func TestIneligible(t *testing.T) {
	api, _, mockHstspreload, mockPreloadlist := mockAPI(0 * time.Second)

	// database.Scan values for testing

	emptyScan := database.Scan{
		Issues: []hstspreload.Issues{emptyIssues},
	}

	warningScan := database.Scan{
		Issues: []hstspreload.Issues{issuesWithWarnings},
	}

	errorScan := database.Scan{
		Issues: []hstspreload.Issues{issuesWithErrors},
	}

	TestEligibleResponses := map[string]hstspreload.Issues{
		"garron.net":   emptyIssues,
		"badssl.com":   issuesWithWarnings,
		"chromium.org": issuesWithErrors,
		"godoc.og":     issuesWithErrors,
		"dev":          issuesWithWarnings,
		"example.com":  issuesWithErrors,
	}
	TestPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "garron.net", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "chromium.org", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false, Policy: preloadlist.Bulk1Year},
		{Name: "godoc.og", Mode: "", IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "dev", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
		{Name: "badssl.com", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "example.com", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
		{Name: "another.com", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.PublicSuffix},
	}}

	expectedScans := map[string]database.Scan{
		"garron.net":   emptyScan,
		"badssl.com":   warningScan,
		"chromium.org": errorScan,
		"godoc.og":     errorScan,
		"dev":          warningScan,
		"example.com":  errorScan,
	}

	expectedPolicies := map[string]string{
		"garron.net":   preloadlist.Bulk18Weeks,
		"chromium.org": preloadlist.Bulk1Year,
		"godoc.og":     preloadlist.Bulk18Weeks,
		"dev":          preloadlist.Bulk1Year,
		"badssl.com":   preloadlist.Bulk18Weeks,
		"example.com":  preloadlist.Bulk1Year,
	}

	mockHstspreload.eligibleResponses = TestEligibleResponses
	mockPreloadlist.list = TestPreloadlist

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}

	api.Update(w, r)
	api.RemoveIneligibleDomains(w, r)

	if w.Code != 200 {
		t.Errorf("HTTP Response Invalid: Status code is not 200")
	}

	states, err := api.database.GetAllIneligibleDomainStates()
	if err != nil {
		t.Fatalf("Couldn't get the states of all domains in the database.")
	}

	for _, state := range states {
		for _, scan := range state.Scans {
			if scan.Match(expectedScans[state.Name].Issues[0]) {
				t.Errorf("Scan field not accurately populated in the database for %s", state.Name)
			}
		}
		if state.Policy != expectedPolicies[state.Name] {
			t.Errorf("Policy field not accurately populated in the database for %s with %s", state.Policy, expectedPolicies[state.Name])
		}
	}
}
