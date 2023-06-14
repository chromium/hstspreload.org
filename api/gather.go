package api

import (
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

type PolicyType int

const (
	bulk18Week PolicyType = iota
	bulk1Year
)

func (pt PolicyType) String() string {
	return []string{"bulk-18-weeks", "bulk-1-year"}[pt]
}

// GetBulk18Weeks reuturns a list of entries from the preloadlist that have "bulk-18-weeks" policy type
func GetBulk18WeeksDomains(list preloadlist.PreloadList) []preloadlist.Entry {
	var domains []preloadlist.Entry

	for _, entry := range list.Entries {
		if entry.Policy == bulk18Week.String() {
			domains = append(domains, entry)
		}
	}

	return domains
}

// GetBulk1Year returns a list of entries from the preloadlist that have "bulk-1-year" policy type
func GetBulk1YearDomains(list preloadlist.PreloadList) []preloadlist.Entry {
	var domains []preloadlist.Entry

	for _, entry := range list.Entries {
		if entry.Policy == bulk1Year.String() {
			domains = append(domains, entry)
		}
	}

	return domains
}
