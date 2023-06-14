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

type Domain string

// GetBulk18Weeks reuturns a list of domains from the preloadlist that are of the "bulk-18-week" policy type
func GetBulk18WeeksDomains(list preloadlist.PreloadList) []Domain {
	var domains []Domain

	for _, entry := range list.Entries {
		if entry.Policy == bulk18Week.String() {
			domains = append(domains, Domain(entry.Name))
		}
	}

	return domains
}

// GetBulk1Year returns a list of domaisn from the preloadlist that are of the "bulk-1-year" policy type
func GetBulk1YearDomains(list preloadlist.PreloadList) []Domain {
	var domains []Domain

	for _, entry := range list.Entries {
		if entry.Policy == bulk1Year.String() {
			domains = append(domains, Domain(entry.Name))
		}
	}

	return domains
}
