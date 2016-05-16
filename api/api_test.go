package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.appspot.com/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

var emptyIssues = hstspreload.Issues{}

var issuesWithWarnings = hstspreload.Issues{
	Warnings: []hstspreload.Issue{{"code", "warning", "message"}},
}

var issuesWithErrors = hstspreload.Issues{
	Errors: []hstspreload.Issue{
		{"code1", "warning1", "message1"},
		{"code2", "warning2", "message2"},
	},
	Warnings: []hstspreload.Issue{{"code", "warning", "message"}},
}

func mockAPI() (api API, mc *database.MockController, h *mockHstspreload, c *mockPreloadlist) {
	db, mc := database.NewMock()
	h = &mockHstspreload{}
	c = &mockPreloadlist{}
	api = API{
		database:    db,
		hstspreload: h,
		preloadlist: c,
	}
	return api, mc, h, c
}

func TestCheckConnection(t *testing.T) {
	api, mc, _, _ := mockAPI()

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
	description string
	mockData    MockData
	failState   int
	handlerFunc http.HandlerFunc
	method      string
	url         string
	wantCode    int
	wantBody    wantBody
}

func TestAPI(t *testing.T) {
	api, mc, h, c := mockAPI()

	pr1 := map[string]hstspreload.Issues{
		"garron.net":  emptyIssues,
		"badssl.com":  issuesWithWarnings,
		"example.com": issuesWithErrors,
	}
	rr1 := map[string]hstspreload.Issues{
		"removable.test":   emptyIssues,
		"unremovable.test": issuesWithErrors,
	}

	pl1 := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{"garron.net", preloadlist.ForceHTTPS, true},
		{"chromium.org", preloadlist.ForceHTTPS, false},
		{"godoc.og", "", true},
	}}

	pl2 := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{"chromium.org", preloadlist.ForceHTTPS, false},
		{"godoc.og", "", true},
	}}

	data1 := MockData{pr1, rr1, pl1}
	data2 := MockData{pr1, rr1, pl2}

	apiTestSequence := []apiTestCase{
		// wrong HTTP method
		{"submit wrong method", data1, failNone, api.Preloadable, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", data1, failNone, api.Removable, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"status wrong method", data1, failNone, api.Status, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"pending wrong method", data1, failNone, api.Pending, "POST", "",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", data1, failNone, api.Submit, "GET", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires POST.\n"}},

		// misc. issues
		{"status wrong method", data1, failNone, api.Status, "GET", "",
			400, wantBody{text: ""}},
		{"status wrong method", data1, failNone, api.Status, "GET", "?domain=",
			400, wantBody{text: ""}},

		// preloadable and removable
		{"preloadable good", data1, failNone, api.Preloadable, "GET", "?domain=garron.net",
			200, wantBody{issues: &emptyIssues}},
		{"preloadable warning", data1, failNone, api.Preloadable, "GET", "?domain=badssl.com",
			200, wantBody{issues: &issuesWithWarnings}},
		{"preloadable error", data1, failNone, api.Preloadable, "GET", "?domain=example.com",
			200, wantBody{issues: &issuesWithErrors}},

		// removable
		{"preloadable good", data1, failNone, api.Removable, "GET", "?domain=removable.test",
			200, wantBody{issues: &emptyIssues}},
		{"preloadable error", data1, failNone, api.Removable, "GET", "?domain=unremovable.test",
			200, wantBody{issues: &issuesWithErrors}},

		// initial
		{"garron.net initial", data1, failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusUnknown}}},
		{"example.com initial", data1, failNone, api.Status, "GET", "?domain=example.com",
			200, wantBody{state: &database.DomainState{
				Name: "example.com", Status: database.StatusUnknown}}},
		{"pending 1", data1, failNone, api.Pending, "GET", "",
			200, wantBody{text: "[\n]\n"}},

		// initial with database failure
		{"pending failure", data1, failDatabase, api.Pending, "GET", "",
			500, wantBody{text: "Internal error: could not retrieve pending list. (forced failure)\n\n"}},
		{"status failure", data1, failDatabase, api.Status, "GET", "?domain=garron.net",
			500, wantBody{text: "Internal error: could not retrieve status. (forced failure)\n\n"}},

		// submit
		{"bad submit", data1, failNone, api.Submit, "POST", "?domain=example.com",
			200, wantBody{issues: &issuesWithErrors}},
		{"submit database failure", data1, failDatabase, api.Submit, "POST", "?domain=garron.net",
			500, wantBody{text: "Internal error: could not get current domain status. (forced failure)\n\n"}},
		{"good submit", data1, failNone, api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &emptyIssues}},

		// pending
		{"pending 2", data1, failNone, api.Pending, "GET", "",
			200, wantBody{text: "[\n    { \"name\": \"garron.net\", \"include_subdomains\": true, \"mode\": \"force-https\" }\n]\n"}},
		{"submit while pending", data1, failNone, api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &hstspreload.Issues{
				Warnings: []hstspreload.Issue{{Code: "server.preload.already_pending"}},
			}}},

		// update
		{"garron.net pending", data1, failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusPending}}},
		{"update chromiumpreload failure", data1, failChromiumpreload, api.Update, "GET", "",
			500, wantBody{text: "Internal error: could not retrieve latest preload list. (forced failure)\n\n"}},
		{"update database failure", data1, failDatabase, api.Update, "GET", "",
			500, wantBody{text: "Internal error: could not retrieve domain names previously marked as preloaded. (forced failure)\n\n"}},
		{"update success", data1, failNone, api.Update, "GET", "",
			200, wantBody{text: "The preload list has 3 entries.\n- # of preloaded HSTS entries: 2\n- # to be added in this update: 2\n- # to be removed this update: 0\nSuccess. 2 domain states updated.\n"}},
		{"pending 3", data1, failNone, api.Pending, "GET", "",
			200, wantBody{text: "[\n]\n"}},

		// after update
		{"submit after preloaded", data1, failNone, api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &hstspreload.Issues{
				Errors: []hstspreload.Issue{{Code: "server.preload.already_preloaded"}},
			}}},
		{"example.com after update", data1, failNone, api.Status, "GET", "?domain=example.com",
			200, wantBody{state: &database.DomainState{
				Name: "example.com", Status: database.StatusUnknown}}},
		{"garron.net after update", data1, failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusPreloaded}}},
		{"chromium.org after update", data1, failNone, api.Status, "GET", "?domain=chromium.org",
			200, wantBody{state: &database.DomainState{
				Name: "chromium.org", Status: database.StatusPreloaded}}},
		{"godoc.org after update", data1, failNone, api.Status, "GET", "?domain=godoc.org",
			200, wantBody{state: &database.DomainState{
				Name: "godoc.org", Status: database.StatusUnknown}}},

		// update with removal
		{"update with removal", data2, failNone, api.Update, "GET", "",
			200, wantBody{text: "The preload list has 2 entries.\n- # of preloaded HSTS entries: 1\n- # to be added in this update: 0\n- # to be removed this update: 1\nSuccess. 1 domain states updated.\n"}},
		{"garron.net after update with removal", data2, failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusRemoved}}},
		{"chromium.org after update with removal", data2, failNone, api.Status, "GET", "?domain=chromium.org",
			200, wantBody{state: &database.DomainState{
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
