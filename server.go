package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"google.golang.org/appengine"

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

	server.HandleFunc("/api/v2/pending", a.Pending)
	server.HandleFunc("/api/v2/update", a.Update)

	fmt.Println("Listening...")
	appengine.Main()
}

func origin(local bool) string {
	if local {
		return "http://localhost:" + port
	}
	return "https://hstspreload.appspot.com"
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
		fmt.Printf("Setting up prod database...")
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
