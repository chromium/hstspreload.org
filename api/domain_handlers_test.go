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

func toAppEngineHttpRequest(r *http.Request) *http.Request {
	r.Header.Set("X-Appengine-Cron", "true")
	r.RemoteAddr = "0.1.0.2"
	return r
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

// TestAddIneligibleDomain tests that IneligibleDomainState Database is populated when the Ineligible endpoint is called.
func TestAddIneligibleDomain(t *testing.T) {
	api, _, mockHstspreload, mockPreloadlist := mockAPI(0 * time.Second)

	// database.Scan values for testing

	errorScan := database.Scan{
		Issues: issuesWithErrors,
	}

	TestEligibleResponses := map[string]hstspreload.Issues{
		"preloaded-bulk-18-weeks-no-issues.test":  emptyIssues,
		"preloaded-bulk-18-weeks-warnings.test":   issuesWithWarnings,
		"preloaded-bulk-1-year-errors.test":       issuesWithErrors,
		"not-preloaded-bulk-18-weeks-errors.test": issuesWithErrors,
		"preloaded-bulk-1-year-warnings":          issuesWithWarnings,
		"preloaded-bulk-18-weeks-errors.test":     issuesWithErrors,
		"preloaded-public-suffix-no-issues":       emptyIssues,
	}
	TestPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "preloaded-bulk-18-weeks-no-issues.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-bulk-1-year-errors.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false, Policy: preloadlist.Bulk1Year},
		{Name: "not-preloaded-bulk-18-weeks-errors.test", Mode: "", IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-bulk-1-year-warnings", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
		{Name: "preloaded-bulk-18-weeks-warnings.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-bulk-18-weeks-errors.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-public-suffix-no-issues", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.PublicSuffix},
	}}

	// only these domains end up being populated
	expectedScans := map[string][]database.Scan{
		"preloaded-bulk-1-year-errors.test":   {errorScan, errorScan},
		"preloaded-bulk-18-weeks-errors.test": {errorScan},
	}

	expectedPolicies := map[string]string{
		"preloaded-bulk-1-year-errors.test":   preloadlist.Bulk1Year,
		"preloaded-bulk-18-weeks-errors.test": preloadlist.Bulk18Weeks,
	}

	mockHstspreload.eligibleResponses = TestEligibleResponses
	mockPreloadlist.list = TestPreloadlist

	// tests that Scan field is appended to if domain is already present in the database
	err := api.database.SetIneligibleDomainStates([]database.IneligibleDomainState{
		{
			Name:   "preloaded-bulk-1-year-errors.test",
			Scans:  []database.Scan{{Issues: issuesWithErrors}},
			Policy: preloadlist.Bulk1Year,
		},
	}, func(format string, args ...interface{}) {})

	if err != nil {
		t.Errorf("Could not Set IneligibleDomain")
	}

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}
	// tests that a non-App Engine http request returns a 403 status code
	invalidW := httptest.NewRecorder()
	invalidW.Body = &bytes.Buffer{}
	if api.RemoveIneligibleDomains(invalidW, r); invalidW.Code != 403 {
		t.Errorf("HTTP Response INvalid: Status code for invalid http request is not 403")
	}
	r = toAppEngineHttpRequest(r)

	api.Update(w, r)
	api.RemoveIneligibleDomains(w, r)

	if w.Code != 200 {
		t.Errorf("HTTP Response Invalid: Status code is not 200, it's %d", w.Code)
	}

	states, err := api.database.GetAllIneligibleDomainStates()
	if err != nil {
		t.Fatalf("Couldn't get the states of all domains in the database.")
	}

	for _, state := range states {
		for i, scan := range state.Scans {
			if !scan.Issues.Match(expectedScans[state.Name][i].Issues) {
				t.Errorf("Scan field not accurately populated in the database for %s", state.Name)
			}
		}
		if state.Policy != expectedPolicies[state.Name] {
			t.Errorf("Policy field not accurately populated in the database for %s with %s", state.Policy, expectedPolicies[state.Name])
		}
	}
}

// TestDeleteIneligibleDomain tests that eligible domains from the IneligibleDomainState database are removed
func TestDeleteIneligibleDomain(t *testing.T) {
	api, _, mockHstspreload, mockPreloadlist := mockAPI(0 * time.Second)

	ineligibleDomains := []database.IneligibleDomainState{
		{
			Name: "preloaded-bulk-1-year-errors.test",
			Scans: []database.Scan{{
				Issues: issuesWithErrors}},
			Policy: preloadlist.Bulk1Year,
		},
		{
			Name: "preloaded-bulk-18-weeks-errors.test",
			Scans: []database.Scan{{
				Issues: issuesWithErrors}},
			Policy: preloadlist.Bulk18Weeks,
		},
	}

	err := api.database.SetIneligibleDomainStates(ineligibleDomains, func(format string, args ...interface{}) {})
	if err != nil {
		t.Errorf("Could not Set IneligibleDomains")
	}

	testEligibleResponses := map[string]hstspreload.Issues{
		"preloaded-bulk-18-weeks-no-issues.test":  emptyIssues,
		"preloaded-bulk-18-weeks-warnings.test":   issuesWithWarnings,
		"preloaded-bulk-1-year-errors.test":       emptyIssues,
		"not-preloaded-bulk-18-weeks-errors.test": issuesWithErrors,
		"preloaded-bulk-1-year-warnings":          issuesWithWarnings,
		"preloaded-bulk-18-weeks-errors.test":     emptyIssues,
		"preloaded-public-suffix-no-issues":       emptyIssues,
	}

	testPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "preloaded-bulk-18-weeks-no-issues.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-bulk-1-year-errors.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
		{Name: "not-preloaded-bulk-18-weeks-errors.test", Mode: "", IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-bulk-1-year-warnings", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
		{Name: "preloaded-bulk-18-weeks-warnings.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-bulk-18-weeks-errors.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-public-suffix-no-issues", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.PublicSuffix},
	}}

	mockHstspreload.eligibleResponses = testEligibleResponses
	mockPreloadlist.list = testPreloadlist

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}
	r = toAppEngineHttpRequest(r)

	api.Update(w, r)
	api.RemoveIneligibleDomains(w, r)

	states, err := api.database.GetAllIneligibleDomainStates()
	if err != nil {
		t.Errorf("Could not get all IneligibleDomains")
	}
	if len(states) != 0 {
		t.Errorf("IneligibleDomain database is not empty")
	}
}

func TestStatusChange(t *testing.T) {
	api, _, mockHstspreload, mockPreloadlist := mockAPI(0 * time.Second)

	testPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "preloaded-errors-ineligible", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-errors-eligible", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-no-errors", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
	}}

	testEligibleResponses := map[string]hstspreload.Issues{
		"preloaded-errors-ineligible": issuesWithErrors,
		"preloaded-errors-eligible":   issuesWithErrors,
		"preloaded-no-errors":         emptyIssues,
	}

	mockHstspreload.eligibleResponses = testEligibleResponses
	mockPreloadlist.list = testPreloadlist

	testIneligibleDomainList := []database.IneligibleDomainState{
		{
			Name: "preloaded-errors-ineligible",
			Scans: []database.Scan{
				{ScanTime: time.Date(2022, time.July, 26, 1, 37, 51, 15, time.UTC)},
			},
			Policy: "Bulk-18-weeks",
		},
		{
			Name: "preloaded-errors-eligible",
			Scans: []database.Scan{
				{ScanTime: time.Now().Add(-time.Hour * 24 * 7)},
			},
			Policy: "Bulk-18-weeks",
		},
		{
			Name: "preloaded-no-errors",
			Scans: []database.Scan{
				{ScanTime: time.Date(2022, time.July, 26, 1, 37, 51, 15, time.UTC)},
			},
			Policy: "Bulk-18-weeks",
		},
	}

	err := api.database.SetIneligibleDomainStates(testIneligibleDomainList, func(format string, args ...interface{}) {})
	if err != nil {
		t.Fatalf("Couldn't set the states of ineligible domains in the database.")
	}

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}
	r = toAppEngineHttpRequest(r)

	api.Update(w, r)
	api.RemoveIneligibleDomains(w, r)

	state, err := api.database.AllDomainStates()
	if err != nil {
		t.Errorf("Couldn't get the state of preloaded-errors-ineligible from the database.")
	}
	for _, s := range state {
		if s.Name != "preloaded-errors-ineligible" && s.Status == database.StatusPendingAutomatedRemoval {
			t.Errorf("Status of %s has been changed", s.Name)
		}
		if s.Name == "preloaded-errors-ineligible" && s.Status != database.StatusPendingAutomatedRemoval {
			t.Errorf("Status has not been changed")
		}
	}
}

// TestDeletionAndStatusChange tests whether the correct domains
// are being deleted and are having their status changed
func TestDeletionAndStatusChange(t *testing.T) {
	api, _, mockHstspreload, mockPreloadlist := mockAPI(0 * time.Second)

	testPreloadlist := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "preloaded-errors-ineligible", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks},
		{Name: "preloaded-no-errors", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year},
	}}

	testEligibleResponses := map[string]hstspreload.Issues{
		"preloaded-errors-ineligible": issuesWithErrors,
		"preloaded-no-errors":         emptyIssues,
	}

	mockHstspreload.eligibleResponses = testEligibleResponses
	mockPreloadlist.list = testPreloadlist

	testIneligibleDomainList := []database.IneligibleDomainState{
		{
			Name: "preloaded-errors-ineligible",
			Scans: []database.Scan{
				{ScanTime: time.Date(2022, time.July, 26, 1, 37, 51, 15, time.UTC)},
			},
			Policy: "Bulk-18-weeks",
		},
		{
			Name: "preloaded-no-errors",
			Scans: []database.Scan{
				{ScanTime: time.Date(2022, time.July, 26, 1, 37, 51, 15, time.UTC)},
			},
			Policy: "Bulk-18-weeks",
		},
	}

	err := api.database.SetIneligibleDomainStates(testIneligibleDomainList, func(format string, args ...interface{}) {})
	if err != nil {
		t.Fatalf("Couldn't set the states of ineligible domains in the database.")
	}

	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	r, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("[%s] %s", "NewRequest Failed", err)
	}
	r = toAppEngineHttpRequest(r)

	api.Update(w, r)
	api.RemoveIneligibleDomains(w, r)

	states, err := api.database.GetAllIneligibleDomainStates()
	if err != nil {
		t.Errorf("Couldn't get all ineligible states from the database.")
	}
	if len(states) != 1 || states[0].Name != "preloaded-errors-ineligible" {
		t.Errorf("Eligible domain has not been deleted")
	}

	// second run
	testEligibleResponses = map[string]hstspreload.Issues{
		"preloaded-errors-ineligible": issuesWithErrors,
		"preloaded-no-errors":         issuesWithErrors,
	}

	mockHstspreload.eligibleResponses = testEligibleResponses

	api.RemoveIneligibleDomains(w, r)

	errState, err := api.database.StateForDomain("preloaded-errors-ineligible")
	if err != nil {
		t.Errorf("Couldn't get the state of preloaded-errors-ineligible from the database.")
	}
	if errState.Status != database.StatusPendingAutomatedRemoval {
		t.Errorf("Status has not been changed")
	}

	state, err := api.database.StateForDomain("preloaded-no-errors")
	if err != nil {
		t.Errorf("Couldn't get the state of preloaded-no-errors from the database.")
	}
	if state.Status == database.StatusPendingAutomatedRemoval {
		t.Errorf("Status changed")
	}
}
