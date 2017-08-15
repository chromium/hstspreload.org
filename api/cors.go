package api

import (
	"net/http"
	"net/url"
)

const (
	corsOriginHeader = "Access-Control-Allow-Origin"
)

// If you have a project that could use client-side API access
// to hstspreload.org, feel free to send a pull request
// to add your domain on GitHub:
// https://github.com/chromium/hstspreload.org/edit/master/api/cors.go
var whitelistedHosts = map[string]bool{
	"mozilla.github.io":       true,
	"observatory.mozilla.org": true,
	"apis.0.me.uk":            true,
	"apis.midnight.0.me.uk":   true,
}

func allowOrigin(clientOrigin string) bool {
	o, err := url.Parse(clientOrigin)
	if err != nil {
		return false
	}

	switch {
	case o.Hostname() == "localhost":
		return true
	case o.Scheme == "https" && whitelistedHosts[o.Hostname()]:
		return true
	default:
		return false
	}
}

func (api API) allowCORS(w http.ResponseWriter, r *http.Request) (cont bool) {
	key := http.CanonicalHeaderKey("Origin")
	clientOrigin := r.Header.Get(key)
	if clientOrigin == "" {
		return true
	}

	if allowOrigin(clientOrigin) {
		w.Header().Set(corsOriginHeader, "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Vary", "Origin")
	} else {
		w.Header().Set(corsOriginHeader, "null")
	}

	return r.Method != http.MethodOptions
}
