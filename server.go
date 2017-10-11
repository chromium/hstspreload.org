package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"google.golang.org/appengine"

	"github.com/chromium/hstspreload.org/api"
	"github.com/chromium/hstspreload.org/database"
)

func main() {
	local := flag.Bool("local", false, "run the server using a local database")
	flag.Parse()

	a, shutdown := mustSetupAPI(*local, mustReadBulkPreloaded())
	defer shutdown()

	server := hstsServer{}

	staticHandler := http.FileServer(http.Dir("frontend"))
	server.Handle("/", staticHandler)
	server.Handle("/version", staticHandler)
	server.Handle("/favicon.ico", staticHandler)
	server.Handle("/static/", staticHandler)

	server.Handle("/search.xml", searchXML(origin(*local)))
	server.HandleFunc("/robots.txt", http.NotFound)

	server.HandleFunc("/api/v2/preloadable", a.Preloadable)
	server.HandleFunc("/api/v2/removable", a.Removable)
	server.HandleFunc("/api/v2/status", a.Status)
	server.HandleFunc("/api/v2/submit", a.Submit)
	server.HandleFunc("/api/v2/remove", a.Remove)

	server.HandleFunc("/api/v2/pending", a.Pending)
	server.HandleFunc("/api/v2/pending-removal", a.PendingRemoval)
	server.HandleFunc("/api/v2/update", a.Update)

	if *local {
		server.HandleFunc("/api/v2/debug/all-states", a.DebugAllStates)
		server.HandleFunc("/api/v2/debug/set-preloaded", a.DebugSetPreloaded)
		server.HandleFunc("/api/v2/debug/set-rejected", a.DebugSetRejected)
	}

	fmt.Printf("Serving from: %s\n", origin(*local))
	appengine.Main()
}

func port() string {
	portStr, valid := os.LookupEnv("PORT")
	if valid {
		return portStr
	}
	// Default port, per https://godoc.org/google.golang.org/appengine#Main
	return "8080"
}

func origin(local bool) string {
	if local {
		return "http://localhost:" + port()
	}
	return "https://hstspreload.org"
}

func mustSetupAPI(local bool, bulkPreloadedEntries map[string]bool) (a api.API, shutdown func() error) {
	var db database.Database

	if local {
		fmt.Printf("Setting up local database...")
		localDB, dbShutdown, err := database.TempLocalDatabase()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		db, shutdown = localDB, dbShutdown
	} else {
		fmt.Printf("Setting up prod database...")
		db = database.ProdDatabase()
		shutdown = func() error { return nil }
	}

	fmt.Printf(" checking database connection...")

	a = api.New(db, bulkPreloadedEntries)
	err := a.CheckConnection()
	exitIfNotNil(err)

	fmt.Println(" done.")
	return a, shutdown
}

func mustReadBulkPreloaded() api.DomainSet {
	file, err := ioutil.ReadFile("static-data/bulk-preloaded.json")
	exitIfNotNil(err)

	var bulkPreloaded api.DomainSet
	err = json.Unmarshal(file, &bulkPreloaded)
	exitIfNotNil(err)

	return bulkPreloaded
}

func exitIfNotNil(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}
