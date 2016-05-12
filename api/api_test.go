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

func mockAPI() (api API, ms *database.MockState, h *mockHstspreload, c *mockChromiumpreload) {
	db, ms := database.NewMock()
	h = &mockHstspreload{}
	c = &mockChromiumpreload{}
	api = API{
		database:        db,
		hstspreload:     h,
		chromiumpreload: c,
	}
	return api, ms, h, c
}

func TestCheckConnection(t *testing.T) {
	api, ms, _, _ := mockAPI()

	if err := api.CheckConnection(); err != nil {
		t.Errorf("%s", err)
	}

	ms.FailCalls = true
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

type apiTestCase struct {
	description string
	handlerFunc http.HandlerFunc
	method      string
	url         string
	wantCode    int
	wantBody    wantBody
}

func TestStatusSubmitPendingUpdate(t *testing.T) {
	api, _, h, c := mockAPI()

	h.preloadableResponses = make(map[string]hstspreload.Issues)
	h.preloadableResponses["garron.net"] = emptyIssues
	h.preloadableResponses["badssl.com"] = issuesWithWarnings
	h.preloadableResponses["example.com"] = issuesWithErrors

	c.list.Entries = []chromiumpreload.PreloadEntry{
		{"garron.net", chromiumpreload.ForceHTTPS, true},
		{"chromium.org", chromiumpreload.ForceHTTPS, false},
		{"godoc.og", "", true},
	}

	wantStatus := func(description string, domain string, status database.PreloadStatus) apiTestCase {
		return apiTestCase{description, api.Status, "GET", "?domain=" + domain,
			200, wantBody{state: &database.DomainState{
				Name:   domain,
				Status: status,
			}},
		}
	}

	apiTestSequence := []apiTestCase{
		// wrong HTTP method
		{"status wrong method", api.Status, "POST", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"pending wrong method", api.Pending, "POST", "",
			405, wantBody{text: "Wrong method. Requires GET.\n"}},
		{"submit wrong method", api.Submit, "GET", "?domain=garron.net",
			405, wantBody{text: "Wrong method. Requires POST.\n"}},

		// initial
		wantStatus("garron.net initial", "garron.net", database.StatusUnknown),
		wantStatus("example.com initial", "example.com", database.StatusUnknown),
		{"pending 1", api.Pending, "GET", "",
			200, wantBody{text: "[\n]\n"}},

		// submit
		{"bad submit", api.Submit, "POST", "?domain=example.com",
			200, wantBody{issues: &issuesWithErrors}},
		{"good submit", api.Submit, "POST", "?domain=garron.net",
			200, wantBody{issues: &emptyIssues}},
		{"pending 2", api.Pending, "GET", "",
			200, wantBody{text: "[\n    { \"name\": \"garron.net\", \"include_subdomains\": true, \"mode\": \"force-https\" }\n]\n"}},

		// update
		wantStatus("garron.net pending", "garron.net", database.StatusPending),
		{"update", api.Update, "GET", "",
			200, wantBody{text: "The preload list has 3 entries.\n- # of preloaded HSTS entries: 2\n- # to be added in this update: 2\n- # to be removed this update: 0\nSuccess. 2 domain states updated.\n"}},
		{"pending 3", api.Pending, "GET", "",
			200, wantBody{text: "[\n]\n"}},

		// after update
		wantStatus("example.com after update", "example.com", database.StatusUnknown),
		wantStatus("garron.net after update", "garron.net", database.StatusPreloaded),
		wantStatus("chromium.org after update", "chromium.org", database.StatusPreloaded),
		wantStatus("godoc.org after update", "godoc.org", database.StatusUnknown),
	}

	for _, tt := range apiTestSequence {
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
