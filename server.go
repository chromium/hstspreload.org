package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chromium/hstspreload.appspot.com/api"
	"github.com/chromium/hstspreload.appspot.com/database"
)

const (
	port = "8080"
)

func main() {
	local := flag.Bool("local", false, "run the server using a local database")
	flag.Parse()

	a, shutdown := mustSetupAPI(*local)
	defer shutdown()

	server := hstsServer{}

	staticHandler := http.FileServer(http.Dir("files"))
	server.Handle("/", staticHandler)
	server.Handle("/version", staticHandler)
	server.Handle("/favicon.ico", staticHandler)
	server.Handle("/static/", staticHandler)

	server.Handle("/search.xml", searchXML(*local))
	server.HandleFunc("/robots.txt", http.NotFound)

	server.HandleFunc("/preloadable", a.Preloadable)
	server.HandleFunc("/removable", a.Removable)
	server.HandleFunc("/status", a.Status)
	server.HandleFunc("/submit", a.Submit)

	server.HandleFunc("/pending", a.Pending)
	server.HandleFunc("/update", a.Update)

	server.HandleFunc("/autocomplete", a.Autocomplete)

	fmt.Println("Listening...")

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
	}
}

func mustSetupAPI(local bool) (a api.API, shutdown func() error) {
	var db database.Database

	if local {
		fmt.Printf("Seting up local database...")
		localDB, dbShutdown, err := database.TempLocalDatabase()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		db, shutdown = localDB, dbShutdown
	} else {
		fmt.Printf("Seting up prod database...")
		db = database.ProdDatabase()
		shutdown = func() error { return nil }
	}

	fmt.Printf(" checking database connection...")

	a = api.New(db)
	if err := a.CheckConnection(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	fmt.Println(" done.")
	return a, shutdown
}

func searchXML(local bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/xml; charset=utf-8")

		var origin string
		if local {
			origin = "http://localhost:" + port
		} else {
			origin = "https://hstspreload.appspot.com"
		}

		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
  <ShortName>HSTS Preload</ShortName>
  <Description>HSTS Preload List Status and Eligibility</Description>
  <Tags>HSTS, HTTPS, security</Tags>
  <Contact>hstspreload@chromium.org</Contact>
  <Url type="text/html" method="GET" template="%s/?domain={searchTerms}"/>
  <Url type="application/x-suggestions+json" method="GET" template="%s/autocomplete?domain={searchTerms}" />
</OpenSearchDescription>`,
			origin,
			origin)

	}
}
