package api

import (
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload.org/database"
)

func (api API) listDomainsWithStatus(w http.ResponseWriter, r *http.Request, status database.PreloadStatus, entryFormat string) {
	if r.Method != "GET" {
		http.Error(w, fmt.Sprintf("Wrong method. Requires GET."), http.StatusMethodNotAllowed)
		return
	}

	names, err := api.database.DomainsWithStatus(status)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve list for status \"%s\". (%s)\n", status, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	fmt.Fprintf(w, "[\n")
	for i, name := range names {
		comma := ","
		if i+1 == len(names) {
			comma = ""
		}

		fmt.Fprintf(w, entryFormat, name, comma)
	}
	fmt.Fprintf(w, "]\n")
}

// Pending returns a list of domains with status "pending".
//
// Example: GET /pending
func (api API) Pending(w http.ResponseWriter, r *http.Request) {
	api.listDomainsWithStatus(w, r, database.StatusPending, `    { "name": "%s", "include_subdomains": true, "mode": "force-https" }%s
`)
}

// PendingRemoval returns a list of domains with status "pending-removal".
//
// Example: GET /pending-removal
func (api API) PendingRemoval(w http.ResponseWriter, r *http.Request) {
	api.listDomainsWithStatus(w, r, database.StatusPendingRemoval, `    "%s"%s
`)
}
