package main

import (
	"fmt"
	"net"
	"net/http"
)

type hstsServer struct{}

func (server hstsServer) Handle(pattern string, handler http.Handler) {
	server.HandleFunc(pattern, handler.ServeHTTP)
}

func (hstsServer) HandleFunc(pattern string, handlerFunc http.HandlerFunc) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if hsts(w, r) {
			handlerFunc(w, r)
		}
	})
}

func isLocalhost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	return err == nil && host == "localhost"
}

// `cont` indicates whether the callee should continue further processing.
func hsts(w http.ResponseWriter, r *http.Request) (cont bool) {
	isHTTPS := r.TLS != nil || maybeAppEngineHTTPS(r)
	if isHTTPS {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	}

	switch {
	case maybeAppEngineCron(r):
		return true
	case (r.Host == "hstspreload.appspot.com"):
		redirectDomain := "hstspreload.appspot.com"
		if isHTTPS {
			redirectDomain = "hstspreload.org"
		}
		u := fmt.Sprintf("https://%s%s", redirectDomain, r.URL.Path)
		http.Redirect(w, r, u, http.StatusMovedPermanently)
		return false
	case isHTTPS, isLocalhost(r.Host):
		return true
	default:
		u := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
		http.Redirect(w, r, u, http.StatusMovedPermanently)
		return false
	}
}

// Note: This can be spoofed when not run on App Engine/Flexible Environment.
func maybeAppEngineCron(r *http.Request) bool {
	return r.Header.Get("X-Appengine-Cron") == "true"
}

// Note: This can be spoofed when not run on App Engine/Flexible Environment.
func maybeAppEngineHTTPS(r *http.Request) bool {
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
