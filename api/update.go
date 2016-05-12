package api

import (
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload.appspot.com/database"
	"github.com/chromium/hstspreload/chromiumpreload"
)

func difference(from []string, take []string) (diff []string) {
	takeSet := make(map[string]bool)
	for _, elem := range take {
		takeSet[elem] = true
	}

	for _, elem := range from {
		if !takeSet[elem] {
			diff = append(diff, elem)
		}
	}

	return diff
}

// Update tells the server to update pending/removed entries based
// on the HSTS preload list source.
//
// Example: GET /update
func (api API) Update(w http.ResponseWriter, r *http.Request) {
	// In order to allow visiting the URL directly in the browser, we allow any method.

	// Get preload list.
	preloadList, listErr := api.chromiumpreload.GetLatest()
	if listErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve latest preload list. (%s)\n",
			listErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	var actualPreload []string
	for _, entry := range preloadList.Entries {
		if entry.Mode == chromiumpreload.ForceHTTPS {
			actualPreload = append(actualPreload, entry.Name)
		}
	}

	// Get domains currently recorded as preloaded.
	databasePreload, dbErr := api.database.DomainsWithStatus(database.StatusPreloaded)
	if dbErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve domain names previously marked as preloaded. (%s)\n",
			dbErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Calculate values that are out of date.
	var updates []database.DomainState

	added := difference(actualPreload, databasePreload)
	for _, name := range added {
		updates = append(updates, database.DomainState{
			Name:   name,
			Status: database.StatusPreloaded,
		})
	}

	removed := difference(databasePreload, actualPreload)
	for _, name := range removed {
		updates = append(updates, database.DomainState{
			Name:   name,
			Status: database.StatusRemoved,
		})
	}

	fmt.Fprintf(w, `The preload list has %d entries.
- # of preloaded HSTS entries: %d
- # to be added in this update: %d
- # to be removed this update: %d
`,
		len(preloadList.Entries),
		len(actualPreload),
		len(added),
		len(removed),
	)

	// Create log function to show progress.
	written := false
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Internal error: Could not create `http.Flusher`.\n", http.StatusInternalServerError)
		return
	}
	logf := func(format string, args ...interface{}) {
		fmt.Fprintf(w, format, args...)
		f.Flush()
		written = true
	}

	// Update the database
	putErr := api.database.PutStates(updates, logf)
	if putErr != nil {
		msg := fmt.Sprintf(
			"Internal error: datastore update failed. (%s)\n",
			putErr,
		)
		if written {
			// The header and part of the body have already been sent, so we
			// can't change the status code anymore.
			fmt.Fprintf(w, msg)
		} else {
			http.Error(w, msg, http.StatusInternalServerError)
		}
		return
	}

	fmt.Fprintf(w, "Success. %d domain states updated.\n", len(updates))
}
