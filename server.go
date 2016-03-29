package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload"
)

func main() {
	staticHandler := http.FileServer(http.Dir("static"))
	http.Handle("/", staticHandler)
	http.Handle("/style.css", staticHandler)
	http.Handle("/index.js", staticHandler)

	http.HandleFunc("/robots.txt", http.NotFound)
	http.HandleFunc("/favicon.ico", http.NotFound)

	http.HandleFunc("/checkdomain/", checkdomain)

	http.HandleFunc("/submit/", submit)
	http.HandleFunc("/clear/", clear)
	http.HandleFunc("/pending", pending)
	http.HandleFunc("/update", update)
	http.HandleFunc("/setmessage", setMessage)
	http.HandleFunc("/setmessages", setMessages)

	http.ListenAndServe(":8080", nil)
}

func checkdomain(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[len("/checkdomain/"):]

	issues := hstspreload.CheckDomain(domain)

	b, err := json.MarshalIndent(hstspreload.MakeSlices(issues), "", "  ")
	if err != nil {
		http.Error(w, "Internal error: could not encode JSON.", 500)
	} else {
		fmt.Fprintf(w, "%s\n", b)
	}
}

func submit(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /submit", 501)
}

func clear(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /clear", 501)
}

func pending(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /pending", 501)
}

func update(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /update", 501)
}

func setMessage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /setMessage", 501)
}

func setMessages(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /setMessages", 501)
}
