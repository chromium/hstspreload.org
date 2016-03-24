package main

import (
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

	http.HandleFunc("/submit/", submit)
	http.HandleFunc("/clear/", clear)
	http.HandleFunc("/pending", pending)
	http.HandleFunc("/update", update)
	http.HandleFunc("/setmessage", setMessage)
	http.HandleFunc("/setmessages", setMessages)

	http.ListenAndServe(":8080", nil)
}


func submit(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[8:]

	err := hstspreload.CheckDomain(domain)

	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err.Error())
	} else {
		fmt.Fprintf(w, "Success!\n")
	}
}

func clear(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /clear", 404)
}

func pending(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /pending", 404)
}

func update(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /update", 404)
}

func setMessage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /setMessage", 404)
}

func setMessages(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unimplemented: /setMessages", 404)
}
