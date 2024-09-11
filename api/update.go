package api

import (
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

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

	domainStates := make(map[string]database.DomainState)
	addDomainStatesWithStatus := func(status database.PreloadStatus) bool {
		domains, err := api.database.StatesWithStatus(status)
		if err != nil {
			msg := fmt.Sprintf("Internal error: could not retrieve domain names previously marked as %s. (%s)\n", status, err)
			http.Error(w, msg, http.StatusInternalServerError)
			return false
		}
		for _, domain := range domains {
			domainStates[domain.Name] = domain
		}
		return true
	}
	// Get domains currently recorded as preloaded, pending removal, or
	// pending automated removal.
	if !addDomainStatesWithStatus(database.StatusPreloaded) ||
		!addDomainStatesWithStatus(database.StatusPendingRemoval) ||
		!addDomainStatesWithStatus(database.StatusPendingAutomatedRemoval) {
		return
	}

	var updates []database.DomainState
	added := 0
	updated := 0
	removed := 0
	for _, entry := range preloadList.Entries {
		if entry.Mode != preloadlist.ForceHTTPS {
			continue
		}
		domainState, found := domainStates[entry.Name]
		if !found {
			// entry is on the preload list but not marked as
			// preloaded, pending removal, or pending automated
			// removal in the database. Mark it as preloaded in the
			// database.
			updates = append(updates, database.EntryToDomainState(entry, database.StatusPreloaded))
			added++
			continue
		}
		delete(domainStates, entry.Name)
		// entry is in both the preload list and in one of the states of
		// preloaded, pending removal, or pending automated removal. If
		// the preload list entry differs from what's in the database,
		// update the database to match.
		if domainState.ToEntry().Equal(entry) {
			continue
		}
		domainState.Policy = entry.Policy
		domainState.IncludeSubDomains = entry.IncludeSubDomains
		updates = append(updates, domainState)
		updated++
	}
	// domainStates now only contains domains that aren't on the preload
	// list. Update their state in the database to mark them as removed.
	for _, domainState := range domainStates {
		domainState.Status = database.StatusRemoved
		domainState.Policy = preloadlist.UnspecifiedPolicyType
		updates = append(updates, domainState)
		removed++
	}

	fmt.Fprintf(w, `The preload list has %d entries.
- # to be added in this update: %d
- # to be updated in this update: %d
- # to be removed this update: %d
`,
		len(preloadList.Entries), added, updated, removed)

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
			fmt.Fprint(w, msg)
		} else {
			http.Error(w, msg, http.StatusInternalServerError)
		}
		return
	}

	fmt.Fprintf(w, "Success. %d domain states updated.\n", len(updates))
}
