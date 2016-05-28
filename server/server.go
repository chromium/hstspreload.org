package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chromium/hstspreload.appspot.com/api"
	"github.com/chromium/hstspreload.appspot.com/db"
)

const (
	port = "8080"
)

func init() {
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

	// 	issues, err := hstspreload.PreloadableDomain("garron.net")

	// 	fmt.Fprintf(w, "test %v %v", issues, err)

	// })

	a, _ := mustSetupAPI(false)
	// defer shutdown()
	http.HandleFunc("/preloadable", a.Preloadable)
	// main()
}

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

	server.HandleFunc("/robots.txt", http.NotFound)

	server.HandleFunc("/preloadable", a.Preloadable)
	server.HandleFunc("/removable", a.Removable)
	server.HandleFunc("/status", a.Status)
	server.HandleFunc("/submit", a.Submit)

	server.HandleFunc("/pending", a.Pending)
	server.HandleFunc("/update", a.Update)

	fmt.Println("Listening...")

	// err := http.ListenAndServe(":"+port, nil)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s", err)
	// }
}

func mustSetupAPI(local bool) (a api.API, shutdown func() error) {
	var database db.Database

	if local {
		// fmt.Printf("Seting up local database...")
		localDB, dbShutdown, err := db.TempLocalDatabase()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		database, shutdown = localDB, dbShutdown
	} else {
		// fmt.Printf("Setting up prod database...")
		database = db.ProdDatabase()
		shutdown = func() error { return nil }
	}

	// fmt.Printf(" checking database connection...")

	a = api.New(database)
	// if err := a.CheckConnection(); err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s", err)
	// 	os.Exit(1)
	// }

	// fmt.Println(" done.")
	return a, shutdown
}
