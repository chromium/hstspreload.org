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

	staticHandler := http.FileServer(http.Dir("files"))
	http.Handle("/", staticHandler)
	http.Handle("/favicon.ico", staticHandler)
	http.Handle("/static/", staticHandler)

	http.HandleFunc("/robots.txt", http.NotFound)

	http.HandleFunc("/preloadable", a.Preloadable)
	http.HandleFunc("/removable", a.Removable)
	http.HandleFunc("/status", a.Status)
	http.HandleFunc("/submit", a.Submit)

	http.HandleFunc("/pending", a.Pending)
	http.HandleFunc("/update", a.Update)

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

	a = api.API{Database: db}
	if err := a.CheckConnection(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	fmt.Println(" done.")
	return a, shutdown
}
