package database

import "time"

// PreloadStatus represents the current status of a domain, e.g. whether it
// is preloaded, pending, etc.
type PreloadStatus string

// Values for PreloadStatus
const (
	StatusUnknown   = "unknown"
	StatusPending   = "pending"
	StatusPreloaded = "preloaded"
	StatusRejected  = "rejected"
	StatusRemoved   = "removed"
)

// DomainState represents the state stored for a domain in the hstspreload
// submission app database.
type DomainState struct {
	// Name is the key in the datastore, so we don't include it as a field
	// in the stored value.
	Name string `datastore:"-" json:"name"`
	// e.g. StatusPending or StatusPreloaded
	Status PreloadStatus `json:"status"`
	// A custom message from the preload list maintainer explaining the
	// current status of the site (usually to explain a StatusRejected).
	Message string `datastore:",noindex" json:"message,omitempty"`
	// The Unix time this domain was last submitted.
	SubmissionDate time.Time `json:"-"`
}
