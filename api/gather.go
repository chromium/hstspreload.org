package api

import (
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

// gatherLists returns a list of domains filtered by Policy types "bulk-18-weeks" and "bulk-1-year"
func gatherLists(list preloadlist.PreloadList) ([]string, []string) {
	var domains18weeks []string
	var domains1year []string

	for _, entry := range list.Entries {
		if entry.Policy == "bulk-18-weeks" {
			domains18weeks = append(domains18weeks, entry.Name)
		}
		if entry.Policy == "bulk-1-year" {
			domains1year = append(domains1year, entry.Name)
		}
	}

	return domains18weeks, domains1year
}
