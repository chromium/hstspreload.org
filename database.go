package main

import (
	"time"

	"golang.org/x/net/context"

	"google.golang.org/cloud/datastore"

	"github.com/chromium/hstspreload/chromiumpreload"
)

const (
	// TODO: Change default to hstspreload and allow a dynamic value.
	projectId = "hstspreload-mvm"
	batchSize = 450
	timeout   = 10 * time.Second

	domainStateKind = "DomainState"
)

type PreloadStatus string

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
	Name chromiumpreload.Domain `datastore:"-" json:"name"`
	// e.g. StatusPending or StatusPreloaded
	Status PreloadStatus `json:"status"`
	// A custom message from the preload list maintainer explaining the
	// current status of the site (usually to explain a StatusException).
	Message string `datastore:",noindex" json:"messsage,omitempty"`
}

// Updates the given domain updates in batches.
// Writes and flushes updates to w.
func putStates(updates []DomainState, statusReport func(format string, args ...interface{})) error {
	if len(updates) == 0 {
		statusReport("No updates.\n")
		return nil
	}

	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := datastore.NewClient(c, projectId)
	if datastoreErr != nil {
		return datastoreErr
	}

	putMulti := func(keys []*datastore.Key, values []*DomainState) error {
		statusReport("Updating %d entries...", len(keys))

		if _, err := client.PutMulti(c, keys, values); err != nil {
			statusReport(" failed.\n")
			return err
		}

		statusReport(" done.\n")
		return nil
	}

	var keys []*datastore.Key
	var values []*DomainState
	for _, state := range updates {
		key := datastore.NewKey(c, domainStateKind, string(state.Name), 0, nil)
		keys = append(keys, key)
		values = append(values, &state)

		if len(keys) >= batchSize {
			if err := putMulti(keys, values); err != nil {
				return err
			}
			keys = keys[:0]
			values = values[:0]
		}
	}

	if err := putMulti(keys, values); err != nil {
		return err
	}
	return nil
}

func putState(update DomainState) error {
	ignoreStatus := func(format string, args ...interface{}) {}
	return putStates([]DomainState{update}, ignoreStatus)
}

func statesForQuery(query *datastore.Query) (states []DomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := datastore.NewClient(c, projectId)
	if datastoreErr != nil {
		return states, datastoreErr
	}

	keys, err := client.GetAll(c, query, &states)
	if err != nil {
		return states, err
	}

	for i, key := range keys {
		state := states[i]
		state.Name = chromiumpreload.Domain(key.Name())
		states[i] = state
	}

	return states, nil
}

func domainsForQuery(query *datastore.Query) (domains []chromiumpreload.Domain, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := datastore.NewClient(c, projectId)
	if datastoreErr != nil {
		return domains, datastoreErr
	}

	keys, err := client.GetAll(c, query.KeysOnly(), nil)
	if err != nil {
		return domains, err
	}

	for _, key := range keys {
		domain := chromiumpreload.Domain(key.Name())
		domains = append(domains, domain)
	}

	return domains, nil
}

// Get the state for the given domain.
// The Name field of `state` will not be set.
func stateForDomain(domain chromiumpreload.Domain) (state DomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := datastore.NewClient(c, projectId)
	if datastoreErr != nil {
		return state, datastoreErr
	}

	key := datastore.NewKey(c, domainStateKind, string(domain), 0, nil)
	getErr := client.Get(c, key, &state)
	if getErr != nil {
		if getErr == datastore.ErrNoSuchEntity {
			return DomainState{Status: StatusUnknown}, nil
		} else {
			return state, getErr
		}
	}

	return state, nil
}

func allDomainStates() (states []DomainState, err error) {
	return statesForQuery(datastore.NewQuery("DomainState"))
}

func domainsWithStatus(status PreloadStatus) (domains []chromiumpreload.Domain, err error) {
	return domainsForQuery(datastore.NewQuery("DomainState").Filter("Status =", string(status)))
}
