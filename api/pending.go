package api

import (
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload.appspot.com/database"
)

// Pending returns a list of domains with status "pending".
//
// Example: GET /pending
func (api API) Pending(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Sprintf("Wrong method. Requires GET."), http.StatusMethodNotAllowed)
		return
	}

	names, err := api.Database.DomainsWithStatus(database.StatusPending)
	if err != nil {
		msg := fmt.Sprintf("Internal error: not convert domain to ASCII. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "[\n")
	for i, name := range names {
		comma := ","
		if i+1 == len(names) {
			comma = ""
		}

		fmt.Fprintf(w, `    { "name": "%s", "include_subdomains": true, "mode": "force-https" }%s
`, name, comma)
	}
	fmt.Fprintf(w, "]\n")
}
