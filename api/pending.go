package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chromium/hstspreload.org/database"
)

// Pending returns a list of domains with status "pending".
//
// Example: GET /pending
func (api API) Pending(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Sprintf("Wrong method. Requires GET."), http.StatusMethodNotAllowed)
		return
	}

	states, err := api.database.DomainStatesWithStatus(database.StatusPending)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve pending list. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	fmt.Fprintf(w, "[\n")
	for i, s := range states {
		data := struct {
			Name           string    `datastore:"-" json:"name"`
			SubmissionDate time.Time `json:"submission_date,omit_empty"`
		}{
			s.Name,
			s.SubmissionDate,
		}

		b, err := json.Marshal(data)
		if err != nil {
			msg := fmt.Sprintf("Internal error: could not format JSON. (%s)\n", err)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		comma := ","
		if i+1 == len(states) {
			comma = ""
		}

		fmt.Fprintf(w, "    %s%s\n", b, comma)
	}
	fmt.Fprintf(w, "]\n")
}
