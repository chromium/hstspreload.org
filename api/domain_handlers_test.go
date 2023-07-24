package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	api, _, _, mockPreloadlist := mockAPI(0 * time.Second)

	// database.Scan values for testing
	formatIssues := database.Scan{
		ScanTime: time.Date(2003, time.April, 11, 6, 30, 2, 143, time.UTC),
		Issues: []hstspreload.Issues{{
			Errors: []hstspreload.Issue{},
		}},
	}

	WWWIssues := database.Scan{
		ScanTime: time.Date(2023, time.July, 24, 1, 38, 25, 98, time.UTC),
		Issues: []hstspreload.Issues{{
			Errors: []hstspreload.Issue{},
		}},
	}

	TestPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: ".garron.net", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "www.chromium.org", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false, Policy: preloadlist.Bulk1Year},
		{Name: "godoc.og", Mode: "", IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: ".dev", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
	}}

	expectedScans := map[string]database.Scan{
		".garron.net":  formatIssues,
		"chromium.org": WWWIssues,
		"www.godoc.og": WWWIssues,
		".dev":         formatIssues,
	}

	expectedPolicies := map[string]string{
		".garron.net":  preloadlist.Bulk18Weeks,
		"chromium.org": preloadlist.Bulk1Year,
		"www.godoc.og": preloadlist.Bulk1Year,
		".dev":         preloadlist.Bulk18Weeks,
	}

	mockPreloadlist.list = TestPreloadlist

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}

	api.Ineligible(w, r)

	if w.Code != 200 {
		t.Errorf("HTTP Response Invalid: Status code is not 200")
	}

	states, err := api.database.GetAllIneligibleDomainStates()
	if err != nil {
		t.Fatalf("Couldn't get the states of all domains in the database.")
	}

	for _, state := range states {
		if !reflect.DeepEqual(state.Scans[0], expectedScans[state.Name]) {
			t.Errorf("Scan field not accurately populated in the database for %s", state.Name)
		}
		if state.Policy != expectedPolicies[state.Name] {
			t.Errorf("Policy field not accurately populated in the database for %s with %s", state.Policy, expectedPolicies[state.Name])
		}
	}
}
