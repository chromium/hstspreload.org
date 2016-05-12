package api

import (
	"testing"

	"github.com/chromium/hstspreload"
)

func TestMockHstspreloadable(t *testing.T) {
	h := mockHstspreload{}

	if _, issues := h.PreloadableDomain("garron.net"); !issues.Match(hstspreload.Issues{}) {
		t.Errorf("Issues should be empty")
	}

	wanted := hstspreload.Issues{Errors: []hstspreload.Issue{{"code", "summary", "message"}}}
	h.preloadableResponses = make(map[string]hstspreload.Issues)
	h.preloadableResponses["garron.net"] = wanted

	if _, issues := h.PreloadableDomain("garron.net"); !issues.Match(wanted) {
		t.Errorf("Issues do not match wanted. %#v", issues)
	}

	if _, issues := h.PreloadableDomain("wikipedia.org"); !issues.Match(hstspreload.Issues{}) {
		t.Errorf("Issues should be empty")
	}

	if _, issues := h.RemovableDomain("garron.net"); !issues.Match(hstspreload.Issues{}) {
		t.Errorf("Issues should be empty")
	}

	wanted = hstspreload.Issues{Warnings: []hstspreload.Issue{{"code", "summary", "message"}}}
	h.removableResponses = make(map[string]hstspreload.Issues)
	h.removableResponses["garron.net"] = wanted

	if _, issues := h.RemovableDomain("garron.net"); !issues.Match(wanted) {
		t.Errorf("Issues do not match wanted. %#v", issues)
	}
}
