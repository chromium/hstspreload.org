package api

import (
	"net/http"

	"github.com/chromium/hstspreload.appspot.com/origin"
)

const (
	corsOriginHeader = "Access-Control-Allow-Origin"
)

// If you have a project that could use client-side API access
// to hstspreload.appspot.com, feel free to send a pull request
// to add your domain on GitHub:
// https://github.com/chromium/hstspreload.appspot.com/edit/master/api/cors.go
var whitelistedHosts = map[string]bool{
	"mozilla.github.io": true,
	"observatory.mozilla.org": true,
	"apis.infinitudecloud.com": true, // RELEASE
	"latenight.apis.infinitudecloud.com": true, // BETA
	"midnight.apis.infinitudecloud.com": true,  // DEVELOPMENT
}

func allowOrigin(clientOrigin string) bool {
	o, err := origin.Parse(clientOrigin)
	if err != nil {
		return false
	}

	switch {
	case o.HostName == "localhost":
		return true
	case o.Scheme == "https" && whitelistedHosts[o.HostName]:
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
