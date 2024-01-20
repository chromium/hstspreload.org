package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"github.com/chromium/hstspreload.org/api"
	"github.com/chromium/hstspreload.org/database"
)

const (
	prodProjectID = "hstspreload"
)

func main() {
	local := flag.Bool("local", false, "run the server using a local database")
	flag.Parse()

	a, shutdown := mustSetupAPI(*local)
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
	server.HandleFunc("/api/v2/pending-automated-removal", a.PendingAutomatedRemoval)

	server.HandleFunc("/api/v2/update", a.Update)

	server.HandleFunc("/api/v2/remove-ineligible-domains", a.RemoveIneligibleDomains)

	if *local {
		server.HandleFunc("/api/v2/debug/all-states", a.DebugAllStates)
		server.HandleFunc("/api/v2/debug/set-preloaded", a.DebugSetPreloaded)
		server.HandleFunc("/api/v2/debug/set-rejected", a.DebugSetRejected)
	}

	server.HandleFunc("/_ah/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	fmt.Printf("Serving from: %s\n", origin(*local))
	fmt.Println(http.ListenAndServe(fmt.Sprintf(":%s", port()), nil))
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

func mustSetupAPI(local bool) (a api.API, shutdown func() error) {
	var db database.Database
	ctx := context.Background()
	logger := log.Default()

	if local {
		logger.Print("Setting up local database...")
		localDB, dbShutdown, err := database.TempLocalDatabase()
		if err != nil {
			logger.Fatalf("Error creating database: %v", err)
		}
		db, shutdown = localDB, dbShutdown
	} else {
		logger.Print("Setting up prod database...")
		db = database.ProdDatabase(prodProjectID)
		logClient, err := logging.NewClient(ctx, prodProjectID)
		if err != nil {
			logger.Fatalf("Failed to create logging client: %v", err)
		}
		logger = logClient.Logger("hstspreload-server").StandardLogger(logging.Info)
		shutdown = func() error { return nil }
	}

	logger.Print(" checking database connection...")

	a = api.New(db, logger)
	err := a.CheckConnection()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	logger.Print("API setup")
	return a, shutdown
}
