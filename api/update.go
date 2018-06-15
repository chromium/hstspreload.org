package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

func difference(from []preloadlist.Entry, take []preloadlist.Entry) (diff []preloadlist.Entry) {
	takeSet := make(map[string]bool)
	for _, elem := range take {
		takeSet[elem.Name] = true
	}

	for _, elem := range from {
		if !takeSet[elem.Name] {
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
	preloadListFetchStart := time.Now()
	preloadList, listErr := api.preloadlist.NewFromLatest()
	preloadListFetchDuration := time.Since(preloadListFetchStart)
	if listErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve latest preload list. (%s)\n",
			listErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	var actualPreload []preloadlist.Entry
	for _, entry := range preloadList.Entries {
		if entry.Mode == preloadlist.ForceHTTPS {
			actualPreload = append(actualPreload, entry)
		}
	}

	// Get domains currently recorded as preloaded.
	preloadedDomainsFetchStart := time.Now()
	preloadedDomains, dbErr := api.database.StatesWithStatus(database.StatusPreloaded)
	preloadedDomainsFetchDuration := time.Since(preloadedDomainsFetchStart)
	if dbErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve domain names previously marked as preloaded. (%s)\n",
			dbErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	var databasePreload []preloadlist.Entry
	for _, ds := range preloadedDomains {
		databasePreload = append(databasePreload, ds.ToEntry())
	}

	// Get domains currently recorded as pending removal.
	pendingRemovalDomains, dbErr := api.database.StatesWithStatus(database.StatusPendingRemoval)
	if dbErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve domain names previously marked as pending removal. (%s)\n",
			dbErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	var databasePendingRemoval []preloadlist.Entry
	for _, ds := range pendingRemovalDomains {
		databasePendingRemoval = append(databasePendingRemoval, ds.ToEntry())
	}

	// Calculate values that are out of date.
	var updates []database.DomainState

	added := difference(difference(actualPreload, databasePreload), databasePendingRemoval)
	for _, entry := range added {
		updates = append(updates, database.DomainState{
			Name:              entry.Name,
			Status:            database.StatusPreloaded,
			IncludeSubDomains: entry.IncludeSubDomains,
		})
	}

	removed := difference(databasePreload, actualPreload)
	for _, entry := range removed {
		updates = append(updates, database.DomainState{
			Name:              entry.Name,
			Status:            database.StatusRemoved,
			IncludeSubDomains: entry.IncludeSubDomains,
		})
	}

	selfRejected := difference(databasePendingRemoval, actualPreload)
	for _, entry := range selfRejected {
		updates = append(updates, database.DomainState{
			Name:              entry.Name,
			Message:           "Domain was added and removed without being preloaded.",
			Status:            database.StatusRejected,
			IncludeSubDomains: entry.IncludeSubDomains,
		})
	}

	fmt.Fprintf(w, `The preload list has %d entries.
- # of preloaded HSTS entries: %d
- # to be added in this update: %d
- # to be removed this update: %d
- # to be self-rejected this update: %d
`,
		len(preloadList.Entries),
		len(actualPreload),
		len(added),
		len(removed),
		len(selfRejected),
	)
	fmt.Fprintf(w, "Time spent fetching preload list: %s\n", preloadListFetchDuration)
	fmt.Fprintf(w, "Time spent loading domains from database: %s\n", preloadedDomainsFetchDuration)

	// Create log function to show progress.
	written := false
	logf := func(format string, args ...interface{}) {
		fmt.Fprintf(w, format, args...)
		// TODO: Reintroduce flushing
		// https://github.com/chromium/hstspreload.org/issues/66
		written = true
	}

	// Update the database.
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
