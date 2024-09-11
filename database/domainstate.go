package database

import (
	"fmt"
	"time"

	"github.com/chromium/hstspreload/chromium/preloadlist"
)

// PreloadStatus represents the current status of a domain, e.g. whether it
// is preloaded, pending, etc.
type PreloadStatus string

// Values for PreloadStatus
const (
	// NOTE: If changing (or adding) these values, also update
	// frontend/static/js/form.js.
	StatusUnknown                 = "unknown"
	StatusPending                 = "pending"
	StatusPreloaded               = "preloaded"
	// StatusRejected is deprecated. StatusRemoved should be used instead.
	StatusRejected                = "rejected"
	StatusRemoved                 = "removed"
	StatusPendingRemoval          = "pending-removal"
	StatusPendingAutomatedRemoval = "pending-automated-removal"
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
	// If this domain is preloaded, this boolean determines whether its descendant
	// domains also are preloaded.
	IncludeSubDomains bool `json:"-"`
	// PolicyType represents the policy under which the domain is a part of the preload list
	Policy preloadlist.PolicyType `json:"-"`
}

// MatchesWanted checks if the fields of `s` match `wanted`.
//
// - Name is always compared.
// - Status is always compared.
// - Message is compared when wanted.Message != nil
// - SubmissionDate is ignored.
func (s DomainState) MatchesWanted(wanted DomainState) bool {
	if wanted.Name != s.Name {
		return false
	}
	if wanted.Status != s.Status {
		return false
	}
	if wanted.Policy != s.Policy {
		return false
	}
	if wanted.Message != "" && wanted.Message != s.Message {
		return false
	}
	return true
}

// Equal checks if the fields of `s` are equal to the fields of `s2`,
// using == for all fields except for SubmissionDate, where Time.Equal is
// used instead. This is a more strict check than MatchesWanted and is
// intended for testing purposes.
func (s DomainState) Equal(s2 DomainState) bool {
	return s.Name == s2.Name && s.Status == s2.Status &&
		s.Message == s2.Message &&
		s.SubmissionDate.Equal(s2.SubmissionDate) &&
		s.IncludeSubDomains == s2.IncludeSubDomains &&
		s.Policy == s2.Policy
}

// ToEntry converts a DomainState to a preloadlist.Entry.
//
// Only the name, preload status, include subdomains boolean and policy is preserved during the conversion.
func (s DomainState) ToEntry() preloadlist.Entry {
	mode := preloadlist.ForceHTTPS
	if s.Status != StatusPreloaded && s.Status != StatusPendingRemoval && s.Status != StatusPendingAutomatedRemoval {
		mode = ""
	}
	return preloadlist.Entry{Name: s.Name, Mode: mode, IncludeSubDomains: s.IncludeSubDomains, Policy: s.Policy}
}

// ToDomainStateWithMessage converts a preloadlist.Entry to a DomainState.
func EntryToDomainStateWithMessage(entry preloadlist.Entry, status PreloadStatus, message string) DomainState {
	return DomainState{Name: entry.Name, Status: status, Message: message, IncludeSubDomains: entry.IncludeSubDomains, Policy: entry.Policy}
}

// EntryToDomainState converts a preloadlist.Entry to a DomainState with no accompanying message.
func EntryToDomainState(entry preloadlist.Entry, status PreloadStatus) DomainState {
	return EntryToDomainStateWithMessage(entry, status, "")
}

func getDomain(states []DomainState, domain string) (DomainState, error) {
	for _, s := range states {
		if s.Name == domain {
			return s, nil
		}
	}
	return DomainState{}, fmt.Errorf("could not find domain state")
}

// MatchWanted checks that:
//
// - All `wanted` domain names are unique.
//
// - `actual` and `wanted` have the same length.
//
// - For every state ws in `wanted` there is a domain s in `actual` such that s.MatchesWanted(ws)
func MatchWanted(actual []DomainState, wanted []DomainState) bool {
	m := make(map[string]bool)
	for _, ws := range wanted {
		if m[ws.Name] {
			return false
		}
		m[ws.Name] = true
	}

	if len(actual) != len(wanted) {
		return false
	}

	for _, ws := range wanted {
		s, err := getDomain(actual, ws.Name)
		if err != nil {
			return false
		}
		if !s.MatchesWanted(ws) {
			return false
		}
	}

	return true
}

func (s DomainState) IsBulk() bool {
	switch s.Policy {
	case preloadlist.Bulk18Weeks:
		return true
	case preloadlist.Bulk1Year:
		return true
	case preloadlist.BulkLegacy:
		return true
	default:
		return false
	}
}

// Protected tells whether a domain is protected from automated removal
func (s DomainState) IsProtected() bool {
	// Pending entries or Bulk preloaded entries are not protected
	if s.Status == StatusPending || s.IsBulk() {
		return false
	}
	return true
}
