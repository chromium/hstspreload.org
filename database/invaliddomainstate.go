package database

import (
	"time"

	"github.com/chromium/hstspreload"
)

type InvalidDomainState struct {
	// Name is the key in the datastore, so we don't include it as a field
	// in the stored value.
	Name string `datastore:"-" json:"name"`
	// The Unix time this domain was scanned and the issues that arose
	Scans []scan `json:"-"` //change to Scans
	//  The policy under which the domain is part of the
	//  preload list. “bulk-18-weeks” or “bulk-1-year”
	Policy string `json:"policy"`
}

type scan struct {
	scanTime time.Time
	issues   []hstspreload.Issues // need to figure out how to change this to Issues object
}
