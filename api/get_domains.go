package api

import (
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

// GetBulk18Weeks reuturns a list of entries from the preloadlist that have "bulk-18-weeks" policy type
func GetBulk18WeeksDomains(list preloadlist.PreloadList) []preloadlist.Entry {
	var domains []preloadlist.Entry

	for _, entry := range list.Entries {
		if entry.Policy == preloadlist.Bulk18Weeks {
			domains = append(domains, entry)
		}
	}

	return domains
}

// GetBulk1Year returns a list of entries from the preloadlist that have "bulk-1-year" policy type
func GetBulk1YearDomains(list preloadlist.PreloadList) []preloadlist.Entry {
	var domains []preloadlist.Entry

	for _, entry := range list.Entries {
		if entry.Policy == preloadlist.Bulk1Year {
			domains = append(domains, entry)
		}
	}

	return domains
}
