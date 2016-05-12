package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.appspot.com/database"
	"github.com/chromium/hstspreload/chromiumpreload"
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

func mockAPI() (api API, mc *database.MockController, h *mockHstspreload, c *mockChromiumpreload) {
	db, mc := database.NewMock()
	h = &mockHstspreload{}
	c = &mockChromiumpreload{}
	api = API{
		database:        db,
		hstspreload:     h,
		chromiumpreload: c,
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

const (
	failDatabase = 1 << iota
	failChromiumpreload
	failNone = 0
)

type apiTestCase struct {
	description string
	failState   int
	handlerFunc http.HandlerFunc
	method      string
	url         string
	wantCode    int
	wantBody    wantBody
}

func TestAPI(t *testing.T) {
	api, mc, h, c := mockAPI()

	h.preloadableResponses = make(map[string]hstspreload.Issues)
	h.preloadableResponses["garron.net"] = emptyIssues
	h.preloadableResponses["badssl.com"] = issuesWithWarnings
	h.preloadableResponses["example.com"] = issuesWithErrors

	h.removableResponses = make(map[string]hstspreload.Issues)
	h.removableResponses["removable.test"] = emptyIssues
	h.removableResponses["unremovable.test"] = issuesWithErrors

	c.list.Entries = []chromiumpreload.PreloadEntry{
		{"garron.net", chromiumpreload.ForceHTTPS, true},
		{"chromium.org", chromiumpreload.ForceHTTPS, false},
		{"godoc.og", "", true},
	}

	apiTestSequence := []apiTestCase{
		// wrong HTTP method
		{"submit wrong method", failNone, api.Preloadable, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", failNone, api.Removable, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"status wrong method", failNone, api.Status, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"pending wrong method", failNone, api.Pending, "POST", "",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", failNone, api.Submit, "GET", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires POST.\n"}},

		// misc. issues
		{"status wrong method", failNone, api.Status, "GET", "",
			400, wantBody{text: ""}},
		{"status wrong method", failNone, api.Status, "GET", "?domain=",
			400, wantBody{text: ""}},

		// preloadable and removable
		{"preloadable good", failNone, api.Preloadable, "GET", "?domain=garron.net",
			200, wantBody{issues: &emptyIssues}},
		{"preloadable warning", failNone, api.Preloadable, "GET", "?domain=badssl.com",
			200, wantBody{issues: &issuesWithWarnings}},
		{"preloadable error", failNone, api.Preloadable, "GET", "?domain=example.com",
			200, wantBody{issues: &issuesWithErrors}},

		// removable
		{"preloadable good", failNone, api.Removable, "GET", "?domain=removable.test",
			200, wantBody{issues: &emptyIssues}},
		{"preloadable error", failNone, api.Removable, "GET", "?domain=unremovable.test",
			200, wantBody{issues: &issuesWithErrors}},

		// initial
		{"garron.net initial", failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusUnknown}}},
		{"example.com initial", failNone, api.Status, "GET", "?domain=example.com",
			200, wantBody{state: &database.DomainState{
				Name: "example.com", Status: database.StatusUnknown}}},
		{"pending 1", failNone, api.Pending, "GET", "",
			200, wantBody{text: "[\n]\n"}},

		// initial with database failure
		{"pending failure", failDatabase, api.Pending, "GET", "",
			500, wantBody{text: "Internal error: could not retrieve pending list. (forced failure)\n\n"}},
		{"status failure", failDatabase, api.Status, "GET", "?domain=garron.net",
			500, wantBody{text: "Internal error: could not retrieve status. (forced failure)\n\n"}},

		// submit
		{"bad submit", failNone, api.Submit, "POST", "?domain=example.com",
			200, wantBody{issues: &issuesWithErrors}},
		{"submit database failure", failDatabase, api.Submit, "POST", "?domain=garron.net",
			500, wantBody{text: "Internal error: could not get current domain status. (forced failure)\n\n"}},
		{"good submit", failNone, api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &emptyIssues}},

		// pending
		{"pending 2", failNone, api.Pending, "GET", "",
			200, wantBody{text: "[\n    { \"name\": \"garron.net\", \"include_subdomains\": true, \"mode\": \"force-https\" }\n]\n"}},
		{"submit while pending", failNone, api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &hstspreload.Issues{
				Warnings: []hstspreload.Issue{{Code: "server.preload.already_pending"}},
			}}},

		// update
		{"garron.net pending", failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusPending}}},
		{"update chromiumpreload failure", failChromiumpreload, api.Update, "GET", "",
			500, wantBody{text: "Internal error: could not retrieve latest preload list. (forced failure)\n\n"}},
		{"update database failure", failDatabase, api.Update, "GET", "",
			500, wantBody{text: "Internal error: could not retrieve domain names previously marked as preloaded. (forced failure)\n\n"}},
		{"update success", failNone, api.Update, "GET", "",
			200, wantBody{text: "The preload list has 3 entries.\n- # of preloaded HSTS entries: 2\n- # to be added in this update: 2\n- # to be removed this update: 0\nSuccess. 2 domain states updated.\n"}},
		{"pending 3", failNone, api.Pending, "GET", "",
			200, wantBody{text: "[\n]\n"}},

		// after update
		{"submit after preloaded", failNone, api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &hstspreload.Issues{
				Errors: []hstspreload.Issue{{Code: "server.preload.already_preloaded"}},
			}}},
		{"example.com after update", failNone, api.Status, "GET", "?domain=example.com",
			200, wantBody{state: &database.DomainState{
				Name: "example.com", Status: database.StatusUnknown}}},
		{"garron.net after update", failNone, api.Status, "GET", "?domain=garron.net",
			200, wantBody{state: &database.DomainState{
				Name: "garron.net", Status: database.StatusPreloaded}}},
		{"chromium.org after update", failNone, api.Status, "GET", "?domain=chromium.org",
			200, wantBody{state: &database.DomainState{
				Name: "chromium.org", Status: database.StatusPreloaded}}},
		{"godoc.org after update", failNone, api.Status, "GET", "?domain=godoc.org",
			200, wantBody{state: &database.DomainState{
				Name: "godoc.org", Status: database.StatusUnknown}}},
	}

	for _, tt := range apiTestSequence {
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
