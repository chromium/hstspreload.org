package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chromium/hstspreload"
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

type preloadedTLDTest struct {
	domain   string
	expected hstspreload.Issues
	method   string
	url      string
}

func TestTLD(t *testing.T) {
	tldTestSequence := []preloadedTLDTest{
		{"example.app",
			hstspreload.Issues{Errors: []hstspreload.Issue{{Code: "domain.preloaded_tld"}}},
			"POST", "?domain=example.app",
		},
		{"example.dev",
			hstspreload.Issues{Errors: []hstspreload.Issue{{Code: "domain.preloaded_tld"}}},
			"POST", "?domain=example.dev",
		},
	}
	for _, tt := range tldTestSequence {
		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}

		_, err := http.NewRequest(tt.method, tt.url, nil)
		if err != nil {
			t.Fatalf("[%s] %s", tt.domain, err)
		}

		var iss hstspreload.Issues
		if err := json.Unmarshal(w.Body.Bytes(), &iss); err != nil {
			t.Fatalf("[%s] %s", tt.domain, err)
		}
		if !iss.Match(*&tt.expected) {
			t.Errorf("[%s] Issues do not match wanted: %#v", tt.domain, iss)
		}

	}
}
