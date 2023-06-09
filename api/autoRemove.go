package api

import (
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload/chromium/preloadlist"
)

func (api API) gatherLists(w http.ResponseWriter, r *http.Request) {
	// Gets a preload list of domains
	prealoadList, listErr := api.preloadlist.NewFromLatest()
	if listErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve latest preload list. (%s)\n",
			listErr,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	var actualPreload []preloadlist.Entry
	for _, entry := range prealoadList.Entries {
		if entry.Mode == preloadlist.ForceHTTPS {
			actualPreload = append(actualPreload, entry)
		}
	}

	// defines domain slices to hold bulk-18-weeks and bulk-1-year domains
	// NOTE THIS IS THE MEANS OF STORING DOMAINS UNTIL WE DEFINE A DATASTORE
	var domains18weeks []string
	var domains1year []string

	// Iterates over the objects and filters them by their policy, if the
	// policy is "custom" we don't do anything
	for _, entry := range actualPreload {
		if entry.Policy == "bulk-18-weeks" {
			domains18weeks = append(domains18weeks, entry.Name)
		}
		if entry.Policy == "bulk-1-year" {
			domains1year = append(domains1year, entry.Name)
		}
	}
}
