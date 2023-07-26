package database

import (
	"time"

	"github.com/chromium/hstspreload"
)

// IneligibleDomainState contains the state about a domain name that 
// is potentially ineligible to remain on the list and is at risk 
// for being removed from the list
type IneligibleDomainState struct {
	// Name is the key in the datastore, so we don't include it as a field
	// in the stored value.
	Name string `datastore:"-" json:"name"`
	// Scans is where information of the checks are stored
	Scans []Scan `json:"-"` 
	//  The policy under which the domain is part of the
	//  preload list. “bulk-18-weeks” or “bulk-1-year”
	Policy string `json:"policy"`
}

// Scan stores the Unix time this domain was scanned and the issues that arose 
type Scan struct {
	ScanTime time.Time
	Issues   hstspreload.Issues 
}
