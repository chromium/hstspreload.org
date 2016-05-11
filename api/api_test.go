package api

import (
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
		t.Errorf("connection should fail")
	}
}
