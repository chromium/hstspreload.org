package api

import (
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

// GetDomainsByPolicyType returns a list of entries from the preloadlist that have the pt policy type
func GetDomainsByPolicyType(list preloadlist.PreloadList, pt preloadlist.PolicyType) []preloadlist.Entry {
	var domains []preloadlist.Entry

	for _, entry := range list.Entries {
		if entry.Policy == pt {
			domains = append(domains, entry)
		}
	}

	return domains
}
