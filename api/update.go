package api

import (
	"fmt"
	"net/http"

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
	preloadList, listErr := api.preloadlist.NewFromLatest()
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
	preloadedDomains, dbErr := api.database.StatesWithStatus(database.StatusPreloaded)
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

// UpdateIncludeSubDomains tells the server to update the IncludeSubDomains property on existing
// datastore entities based on the HSTS preload list source.
//
// The datastore Store entities used to not have this property, as a result this property is implicitly
// false for all entities. This action loads the preload list from the source, and explicitly sets
// IncludeSubDomains on entities with preloaded status if its corresponding Entry in the preload
// list has include_subdomains = true.
//
// This action should only be used during data transition. After all existing entities have their
// IncludeSubDomains property set correctly, this handler function should be deleted. The regular
// Update function has been updated to set the IncludeSubDomains property correctly when updating
// datastore.
//
// Example: GET /update-includesubdomains
//
// TODO: delte this function once data migration is completed.
func (api API) UpdateIncludeSubDomains(w http.ResponseWriter, r *http.Request) {
	// In order to allow visiting the URL directly in the browser, we allow any method.

	// Get preload list.
	preloadList, listErr := api.preloadlist.NewFromLatest()
	if listErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve latest preload list. (%s)\n",
			listErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	actualPreload := make(map[string]preloadlist.Entry)
	for _, entry := range preloadList.Entries {
		if entry.Mode == preloadlist.ForceHTTPS {
			actualPreload[entry.Name] = entry
		}
	}

	// Get domains currently recorded as preloaded.
	preloadedDomains, dbErr := api.database.StatesWithStatus(database.StatusPreloaded)
	if dbErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve domain names previously marked as preloaded. (%s)\n",
			dbErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Find entities in the database that need their property updated.
	var updates []database.DomainState
	for _, ds := range preloadedDomains {
		if actualPreload[ds.Name].IncludeSubDomains && !ds.IncludeSubDomains {
			ds.IncludeSubDomains = true
			updates = append(updates, ds)
		}
	}

	fmt.Fprintf(w, `The database has %d entries.
- # of entries to be updated: %d
`,
		len(preloadedDomains),
		len(updates),
	)

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
