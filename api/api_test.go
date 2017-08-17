package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

var emptyIssues = hstspreload.Issues{}

var issuesWithWarnings = hstspreload.Issues{
	Warnings: []hstspreload.Issue{{Code: "code", Summary: "warning", Message: "message"}},
}

var issuesWithErrors = hstspreload.Issues{
	Errors: []hstspreload.Issue{
		{Code: "code1", Summary: "warning1", Message: "message1"},
		{Code: "code2", Summary: "warning2", Message: "message2"},
	},
	Warnings: []hstspreload.Issue{{Code: "code", Summary: "warning", Message: "message"}},
}

var issuesRemovableProtected = hstspreload.Issues{
	Errors: []hstspreload.Issue{
		{Code: "server.removable.protected"},
	},
}

var issuesRemoveProtected = hstspreload.Issues{
	Errors: []hstspreload.Issue{
		{Code: "server.remove.protected"},
	},
}

// TODO: just reach directly into the API fields in tests.
func mockAPI(cacheDuration time.Duration) (api API, mc *database.MockController, h *mockHstspreload, c *mockPreloadlist, e map[string]bool) {
	db, mc := database.NewMock()
	h = &mockHstspreload{}
	c = &mockPreloadlist{}
	e = make(map[string]bool)
	api = API{
		database:      db,
		hstspreload:   h,
		preloadlist:   c,
		bulkPreloaded: e,
		cache:         cacheWithDuration(cacheDuration),
	}
	return api, mc, h, c, e
}

func TestCheckConnection(t *testing.T) {
	api, mc, _, _, _ := mockAPI(0 * time.Second)

	if err := api.CheckConnection(); err != nil {
		t.Errorf("%s", err)
	}

	mc.FailCalls = true
	if err := api.CheckConnection(); err == nil {
		t.Error("connection should fail")
	}
}

// Any non-zero values are considered wanted.
type wantBody struct {
	text   string
	state  *database.DomainState
	issues *hstspreload.Issues
}

type MockData struct {
	preloadableResponses map[string]hstspreload.Issues
	removableResponses   map[string]hstspreload.Issues
	preloadlist          preloadlist.PreloadList
}

const (
	failDatabase = 1 << iota
	failChromiumpreload
	failNone = 0
)

type apiTestCase struct {
	description     string
	mockData        MockData
	failState       int
	handlerFunc     http.HandlerFunc
	method          string
	url             string
	wantCode        int
	wantContentType string
	wantBody        wantBody
}

func TestAPI(t *testing.T) {
	api, mc, h, c, e := mockAPI(0 * time.Second)

	e["removal-preloaded-bulk-eligible.test"] = true
	e["removal-not-preloaded-bulk-eligible.test"] = true
	e["removal-preloaded-bulk-ineligible.test"] = true

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

	pl1 := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "garron.net", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true},
		{Name: "chromium.org", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false},
		{Name: "removal-preloaded-bulk-eligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true},
		{Name: "removal-preloaded-not-bulk-eligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true},
		{Name: "removal-preloaded-bulk-ineligible.test", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: true},
		{Name: "godoc.og", Mode: "", IncludeSubDomains: true},
	}}

	pl2 := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "chromium.org", Mode: preloadlist.ForceHTTPS, IncludeSubDomains: false},
		{Name: "godoc.og", Mode: "", IncludeSubDomains: true},
	}}

	data1 := MockData{pr1, rr1, pl1}
	data2 := MockData{pr1, rr1, pl2}

	jsonContentType := "application/json; charset=utf-8"
	textContentType := "text/plain; charset=utf-8" // Errors

	apiTestSequence := []apiTestCase{
		// wrong HTTP method
		{"submit wrong method", data1, failNone, api.Preloadable, "POST", "?domain=garron.net",
			405, textContentType, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", data1, failNone, api.Removable, "POST", "?domain=garron.net",
			405, textContentType, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"status wrong method", data1, failNone, api.Status, "POST", "?domain=garron.net",
			405, textContentType, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"pending wrong method", data1, failNone, api.Pending, "POST", "",
			405, textContentType, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", data1, failNone, api.Submit, "GET", "?domain=garron.net",
			405, textContentType, wantBody{text: "Wrong method. Requires POST.\n"}},

		// misc. issues
		{"status wrong method", data1, failNone, api.Status, "GET", "",
			400, textContentType, wantBody{text: ""}},
		{"status wrong method", data1, failNone, api.Status, "GET", "?domain=",
			400, textContentType, wantBody{text: ""}},

		// preloadable and removable
		{"preloadable good", data1, failNone, api.Preloadable, "GET", "?domain=garron.net",
			200, jsonContentType, wantBody{issues: &emptyIssues}},
		{"preloadable warning", data1, failNone, api.Preloadable, "GET", "?domain=badssl.com",
			200, jsonContentType, wantBody{issues: &issuesWithWarnings}},
		{"preloadable error", data1, failNone, api.Preloadable, "GET", "?domain=example.com",
			200, jsonContentType, wantBody{issues: &issuesWithErrors}},
		// initial
		{"garron.net initial", data1, failNone, api.Status, "GET", "?domain=garron.net",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusUnknown}}},
		{"example.com initial", data1, failNone, api.Status, "GET", "?domain=example.com",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "example.com", Status: database.StatusUnknown}}},
		{"pending 1", data1, failNone, api.Pending, "GET", "",
			200, jsonContentType, wantBody{text: "[\n]\n"}},

		// initial with database failure
		{"pending failure", data1, failDatabase, api.Pending, "GET", "",
			500, textContentType, wantBody{text: "Internal error: could not retrieve list for status \"pending\". (forced failure)\n\n"}},
		{"status failure", data1, failDatabase, api.Status, "GET", "?domain=garron.net",
			500, textContentType, wantBody{text: "Internal error: could not retrieve status. (forced failure)\n\n"}},

		// submit
		{"bad submit", data1, failNone, api.Submit, "POST", "?domain=example.com",
			200, jsonContentType, wantBody{issues: &issuesWithErrors}},
		{"submit database failure", data1, failDatabase, api.Submit, "POST", "?domain=garron.net",
			500, textContentType, wantBody{text: "Internal error: could not get current domain status. (forced failure)\n\n"}},
		{"good submit", data1, failNone, api.Submit, "POST", "?domain=garron.net",
			200, jsonContentType, wantBody{issues: &emptyIssues}},

		// pending
		{"pending 2", data1, failNone, api.Pending, "GET", "",
			200, jsonContentType, wantBody{text: "[\n    { \"name\": \"garron.net\", \"include_subdomains\": true, \"mode\": \"force-https\" }\n]\n"}},
		{"submit while pending", data1, failNone, api.Submit, "POST", "?domain=garron.net",
			200, jsonContentType, wantBody{issues: &hstspreload.Issues{
				Warnings: []hstspreload.Issue{{Code: "server.preload.already_pending"}},
			}}},

		// update
		{"garron.net pending", data1, failNone, api.Status, "GET", "?domain=garron.net",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusPending}}},
		{"update chromiumpreload failure", data1, failChromiumpreload, api.Update, "GET", "",
			500, textContentType, wantBody{text: "Internal error: could not retrieve latest preload list. (forced failure)\n\n"}},
		{"update database failure", data1, failDatabase, api.Update, "GET", "",
			500, textContentType, wantBody{text: "Internal error: could not retrieve domain names previously marked as preloaded. (forced failure)\n\n"}},
		{"update success", data1, failNone, api.Update, "GET", "",
			200, textContentType, wantBody{text: "The preload list has 6 entries.\n- # of preloaded HSTS entries: 5\n- # to be added in this update: 5\n- # to be removed this update: 0\nSuccess. 5 domain states updated.\n"}},
		{"pending 3", data1, failNone, api.Pending, "GET", "",
			200, jsonContentType, wantBody{text: "[\n]\n"}},

		// create removable pending
		{"create removable pending eligible", data1, failNone, api.Submit, "POST", "?domain=removal-pending-eligible.test",
			200, jsonContentType, wantBody{issues: &emptyIssues}},
		{"create removable pending ineligible", data1, failNone, api.Submit, "POST", "?domain=removal-pending-ineligible.test",
			200, jsonContentType, wantBody{issues: &emptyIssues}},

		// removable
		{"removable preloaded-bulk-eligible", data1, failNone, api.Removable, "GET", "?domain=removal-preloaded-bulk-eligible.test",
			200, jsonContentType, wantBody{issues: &emptyIssues}},
		{"removable preloaded-not-bulk-eligible", data1, failNone, api.Removable, "GET", "?domain=removal-preloaded-not-bulk-eligible.test",
			200, jsonContentType, wantBody{issues: &issuesRemovableProtected}},
		{"removable preloaded-bulk-ineligible", data1, failNone, api.Removable, "GET", "?domain=removal-preloaded-bulk-ineligible.test",
			200, jsonContentType, wantBody{issues: &issuesWithErrors}},
		{"removable pending-eligible", data1, failNone, api.Removable, "GET", "?domain=removal-pending-eligible.test",
			200, jsonContentType, wantBody{issues: &emptyIssues}},
		{"removable pending-ineligible", data1, failNone, api.Removable, "GET", "?domain=removal-pending-ineligible.test",
			200, jsonContentType, wantBody{issues: &issuesWithErrors}},

		// remove
		{"remove preloaded-bulk-eligible", data1, failNone, api.Remove, "POST", "?domain=removal-preloaded-bulk-eligible.test",
			200, jsonContentType, wantBody{issues: &emptyIssues}},
		{"remove preloaded-not-bulk-eligible", data1, failNone, api.Remove, "POST", "?domain=removal-preloaded-not-bulk-eligible.test",
			200, jsonContentType, wantBody{issues: &issuesRemoveProtected}},
		{"remove preloaded-bulk-ineligible", data1, failNone, api.Remove, "POST", "?domain=removal-preloaded-bulk-ineligible.test",
			200, jsonContentType, wantBody{issues: &issuesWithErrors}},
		{"remove pending-eligible", data1, failNone, api.Remove, "POST", "?domain=removal-pending-eligible.test",
			200, jsonContentType, wantBody{issues: &emptyIssues}},
		{"remove pending-ineligible", data1, failNone, api.Remove, "POST", "?domain=removal-pending-ineligible.test",
			200, jsonContentType, wantBody{issues: &issuesWithErrors}},

		// Check removals
		{"remove preloaded-bulk-eligible", data1, failNone, api.Status, "GET", "?domain=removal-preloaded-bulk-eligible.test",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "removal-preloaded-bulk-eligible.test", Status: database.StatusPendingRemoval}}},
		{"remove preloaded-not-bulk-eligible", data1, failNone, api.Status, "GET", "?domain=removal-preloaded-not-bulk-eligible.test",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "removal-preloaded-not-bulk-eligible.test", Status: database.StatusPreloaded}}},
		{"remove preloaded-bulk-ineligible", data1, failNone, api.Status, "GET", "?domain=removal-preloaded-bulk-ineligible.test",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "removal-preloaded-bulk-ineligible.test", Status: database.StatusPreloaded}}},
		{"remove pending-eligible", data1, failNone, api.Status, "GET", "?domain=removal-pending-eligible.test",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "removal-pending-eligible.test", Status: database.StatusPendingRemoval}}},
		{"remove pending-ineligible", data1, failNone, api.Status, "GET", "?domain=removal-pending-ineligible.test",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "removal-pending-ineligible.test", Status: database.StatusPending}}},

		// after update
		{"submit after preloaded", data1, failNone, api.Submit, "POST", "?domain=garron.net",
			200, jsonContentType, wantBody{issues: &hstspreload.Issues{
				Errors: []hstspreload.Issue{{Code: "server.preload.already_preloaded"}},
			}}},
		{"example.com after update", data1, failNone, api.Status, "GET", "?domain=example.com",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "example.com", Status: database.StatusUnknown}}},
		{"garron.net after update", data1, failNone, api.Status, "GET", "?domain=garron.net",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusPreloaded}}},
		{"chromium.org after update", data1, failNone, api.Status, "GET", "?domain=chromium.org",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "chromium.org", Status: database.StatusPreloaded}}},
		{"godoc.org after update", data1, failNone, api.Status, "GET", "?domain=godoc.org",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "godoc.org", Status: database.StatusUnknown}}},

		// update with removal
		{"update with removal", data2, failNone, api.Update, "GET", "",
			200, textContentType, wantBody{text: "The preload list has 2 entries.\n- # of preloaded HSTS entries: 1\n- # to be added in this update: 0\n- # to be removed this update: 3\nSuccess. 3 domain states updated.\n"}},
		{"garron.net after update with removal", data2, failNone, api.Status, "GET", "?domain=garron.net",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusRemoved}}},
		{"chromium.org after update with removal", data2, failNone, api.Status, "GET", "?domain=chromium.org",
			200, jsonContentType, wantBody{state: &database.DomainState{
				Name: "chromium.org", Status: database.StatusPreloaded}}},
	}

	for _, tt := range apiTestSequence {
		h.preloadableResponses = tt.mockData.preloadableResponses
		h.removableResponses = tt.mockData.removableResponses
		c.list = tt.mockData.preloadlist

		mc.FailCalls = (tt.failState & failDatabase) != 0
		c.failCalls = (tt.failState & failChromiumpreload) != 0

		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}

		r, err := http.NewRequest(tt.method, tt.url, nil)
		if err != nil {
			t.Fatalf("[%s] %s", tt.description, err)
		}

		tt.handlerFunc(w, r)

		contentType := w.HeaderMap.Get(http.CanonicalHeaderKey("Content-Type"))
		if contentType != tt.wantContentType {
			t.Errorf("[%s] Wrong content type: %s", tt.description, contentType)
		}

		if w.Code != tt.wantCode {
			t.Errorf("[%s] Status code does not match wanted: %d", tt.description, w.Code)
		}

		if tt.wantBody.text != "" {
			text := w.Body.String()
			if text != tt.wantBody.text {
				t.Errorf("[%s] Body text does not match wanted: %#v", tt.description, text)
			}
		}

		if tt.wantBody.state != nil {
			var s database.DomainState
			if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
				t.Fatalf("[%s] %s", tt.description, err)
			}
			if !s.MatchesWanted(*tt.wantBody.state) {
				t.Errorf("[%s] Domain state does not match wanted: %#v", tt.description, s)
			}
		}

		if tt.wantBody.issues != nil {
			var iss hstspreload.Issues
			if err := json.Unmarshal(w.Body.Bytes(), &iss); err != nil {
				t.Fatalf("[%s] %s", tt.description, err)
			}
			if !iss.Match(*tt.wantBody.issues) {
				t.Errorf("[%s] Issues do not match wanted: %#v", tt.description, iss)
			}
		}
	}
}

func TestCORS(t *testing.T) {
	api, _, _, _, _ := mockAPI(0 * time.Second)

	cases := []struct {
		handlerName  string
		handlerFunc  http.HandlerFunc
		method       string
		clientOrigin string
		wantCORS     string
	}{
		// Handlers that should allow CORS.
		{"Preloadable", api.Preloadable, http.MethodGet, "", ""},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://example.com", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://example.com:80", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://example.com:443", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "https://example.com", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "https://example.com:80", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "https://example.com:443", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://localhost", "*"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://localhost:8080", "*"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://mozilla.github.io", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://mozilla.github.io:80", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "http://mozilla.github.io:443", "null"},
		{"Preloadable", api.Preloadable, http.MethodGet, "https://mozilla.github.io", "*"},
		{"Preloadable", api.Preloadable, http.MethodGet, "https://mozilla.github.io:80", "*"},
		{"Preloadable", api.Preloadable, http.MethodGet, "https://mozilla.github.io:443", "*"},
		{"Preloadable", api.Preloadable, http.MethodOptions, "http://localhost", "*"},
		{"Preloadable", api.Preloadable, http.MethodOptions, "http://example.com", "null"},
		{"Preloadable", api.Preloadable, http.MethodOptions, "https://example.com", "null"},
		{"Preloadable", api.Preloadable, http.MethodOptions, "http://mozilla.github.io", "null"},
		{"Preloadable", api.Preloadable, http.MethodOptions, "https://mozilla.github.io", "*"},
		{"Preloadable", api.Preloadable, http.MethodPost, "https://mozilla.github.io", "*"},
		{"Status", api.Status, http.MethodGet, "http://localhost:8080", "*"},
		{"Status", api.Status, http.MethodGet, "http://example.com", "null"},
		{"Status", api.Status, http.MethodGet, "https://example.com", "null"},
		{"Status", api.Status, http.MethodGet, "http://mozilla.github.io", "null"},
		{"Status", api.Status, http.MethodGet, "https://mozilla.github.io", "*"},
		{"Status", api.Status, http.MethodOptions, "http://localhost:8080", "*"},
		{"Status", api.Status, http.MethodOptions, "http://example.com", "null"},
		{"Status", api.Status, http.MethodOptions, "https://example.com", "null"},
		{"Status", api.Status, http.MethodOptions, "http://mozilla.github.io", "null"},
		{"Status", api.Status, http.MethodOptions, "https://mozilla.github.io", "*"},
		// Handlers that should not allow CORS.
		{"Removable", api.Removable, http.MethodGet, "http://localhost:8080", ""},
		{"Removable", api.Removable, http.MethodGet, "http://example.com", ""},
		{"Removable", api.Removable, http.MethodGet, "https://example.com", ""},
		{"Removable", api.Removable, http.MethodGet, "http://mozilla.github.io", ""},
		{"Removable", api.Removable, http.MethodGet, "https://mozilla.github.io", ""},
		{"Removable", api.Removable, http.MethodOptions, "https://mozilla.github.io", ""},
		{"Submit", api.Submit, http.MethodGet, "https://mozilla.github.io", ""},
		{"Submit", api.Submit, http.MethodOptions, "https://mozilla.github.io", ""},
		{"Pending", api.Pending, http.MethodGet, "https://mozilla.github.io", ""},
		{"Pending", api.Pending, http.MethodOptions, "https://mozilla.github.io", ""},
		{"Update", api.Update, http.MethodGet, "https://mozilla.github.io", ""},
		{"Update", api.Update, http.MethodOptions, "https://mozilla.github.io", ""},
	}

	for _, tt := range cases {
		r, err := http.NewRequest(tt.method, "", nil)
		if err != nil {
			t.Fatalf("%s", err)
		}
		r.Header.Set("Origin", tt.clientOrigin)

		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}

		tt.handlerFunc(w, r)

		key := http.CanonicalHeaderKey(corsOriginHeader)
		actual := w.Header().Get(key)
		if tt.wantCORS != actual {
			t.Errorf(
				"[%s][%s][%s] CORS header `%s` does not match expected value `%s`.",
				tt.handlerName,
				tt.method,
				tt.clientOrigin,
				actual,
				tt.wantCORS,
			)
		}
	}
}
