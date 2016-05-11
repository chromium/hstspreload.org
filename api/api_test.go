package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chromium/hstspreload.appspot.com/database"
)

func TestCheckConnection(t *testing.T) {
	ms := database.MockState{}
	a := API{database.Mock{State: &ms}}
	if err := a.CheckConnection(); err != nil {
		t.Errorf("%s", err)
	}

	ms.FailCalls = true
	if err := a.CheckConnection(); err == nil {
		t.Error("connection should fail")
	}
}

func TestStatus(t *testing.T) {
	ms := database.MockState{}
	a := API{database.Mock{State: &ms}}

	w := httptest.NewRecorder()

	r, err := http.NewRequest("GET", "?domain=garron.net", nil)
	if err != nil {
		t.Fatal(err)
	}

	b := &bytes.Buffer{}
	w.Body = b

	a.Status(w, r)

	s := database.DomainState{}
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}

	if s.Name != "garron.net" {
		t.Errorf("Wrong name: %s", s.Name)
	}
	if s.Status != database.StatusUnknown {
		t.Errorf("Wrong status: %s", s.Status)
	}

}
