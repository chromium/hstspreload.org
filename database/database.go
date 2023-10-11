package database

import (
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/datastore"
	"github.com/chromium/hstspreload.org/database/gcd"
)

const (
	localProjectID = "hstspreload-local"
	prodProjectID  = "hstspreload"

	batchSize = 450
	timeout   = 90 * time.Second

	domainStateKind           = "DomainState"
	ineligibleDomainStateKind = "IneligibleDomainState"
)

// A Database is an abstraction over Datastore with hstspreload-specific
// database functions.
type Database interface {
	PutStates([]DomainState, func(string, ...interface{})) error
	PutState(DomainState) error
	StateForDomain(string) (DomainState, error)
	StatesForDomains([]string) ([]DomainState, error)
	AllDomainStates() ([]DomainState, error)
	StatesWithStatus(PreloadStatus) ([]DomainState, error)
	GetIneligibleDomainStates(domains []string) (states []IneligibleDomainState, err error)
	SetIneligibleDomainStates(updates []IneligibleDomainState, logf func(format string, args ...interface{})) error
	DeleteIneligibleDomainStates(domains []string) (err error)
	GetAllIneligibleDomainStates() (states []IneligibleDomainState, err error)
}

// DatastoreBacked is a database backed by a gcd.Backend.
type DatastoreBacked struct {
	backend   gcd.Backend
	projectID string
}

// TempLocalDatabase spin up an local in-memory database based
// on a Google Cloud Datastore emulator.
func TempLocalDatabase() (db DatastoreBacked, shutdown func() error, err error) {
	backend, shutdown, err := gcd.NewLocalBackend()
	return DatastoreBacked{backend, localProjectID}, shutdown, err
}

// ProdDatabase gives a Database that will call out to
// the real production instance of Google Cloud Datastore
func ProdDatabase() (db DatastoreBacked) {
	return DatastoreBacked{gcd.NewProdBackend(), prodProjectID}
}

var blackholeLogf = func(format string, args ...interface{}) {}

// PutStates updates the given domain updates in batches.
// Writes updates to logf in real-time.
func (db DatastoreBacked) PutStates(updates []DomainState, logf func(format string, args ...interface{})) error {
	if len(updates) == 0 {
		logf("No updates.\n")
		return nil
	}

	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return datastoreErr
	}

	putMulti := func(keys []*datastore.Key, values []DomainState) error {
		logf("Updating %d entries...", len(keys))

		if _, err := client.PutMulti(c, keys, values); err != nil {
			logf(" failed.\n")
			return err
		}

		logf(" done.\n")
		return nil
	}

	var keys []*datastore.Key
	var values []DomainState
	for _, state := range updates {
		key := datastore.NameKey(domainStateKind, string(state.Name), nil)
		keys = append(keys, key)
		values = append(values, state)

		if len(keys) >= batchSize {
			if err := putMulti(keys, values); err != nil {
				return err
			}
			keys = keys[:0]
			values = values[:0]
		}
	}

	return putMulti(keys, values)
}

// PutState is a convenience version of PutStates for a single domain.
func (db DatastoreBacked) PutState(update DomainState) error {
	return db.PutStates([]DomainState{update}, blackholeLogf)
}

// statesForQuery returns the states for the given datastore query.
func (db DatastoreBacked) statesForQuery(query *datastore.Query) (states []DomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return states, datastoreErr
	}

	keys, err := client.GetAll(c, query, &states)
	if err != nil {
		return states, err
	}

	for i, key := range keys {
		state := states[i]
		state.Name = key.Name
		states[i] = state
	}

	return states, nil
}

// StateForDomain get the state for the given domain.
// Note that the Name field of `state` will not be set.
func (db DatastoreBacked) StateForDomain(domain string) (state DomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return state, datastoreErr
	}

	key := datastore.NameKey(domainStateKind, domain, nil)
	getErr := client.Get(c, key, &state)
	if getErr != nil {
		if getErr == datastore.ErrNoSuchEntity {
			return DomainState{Status: StatusUnknown}, nil
		}
		return state, getErr
	}

	return state, nil
}

// StatesForDomains returns the domains states for the given domains
func (db DatastoreBacked) StatesForDomains(domains []string) (states []DomainState, err error) {
	// Set up datastore context
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return nil, datastoreErr
	}

	getDomainStates := func(keys []*datastore.Key) ([]DomainState, error) {
		domainStates := make([]DomainState, len(keys))
		if err := client.GetMulti(c, keys, domainStates); err != nil {
			return nil, err
		}
		for i := range domainStates {
			domainStates[i].Name = keys[i].Name
		}
		return domainStates, nil
	}

	var keys []*datastore.Key
	var res []DomainState
	for _, domain := range domains {
		key := datastore.NameKey(domainStateKind, domain, nil)
		keys = append(keys, key)
		if len(keys) >= batchSize {
			var tempStates []DomainState
			if tempStates, err = getDomainStates(keys); err != nil {
				return nil, err
			}
			res = append(res, tempStates...)
			keys = keys[:0]

		}
	}

	var getStates []DomainState
	if getStates, err = getDomainStates(keys); err != nil {
		return nil, err
	}
	return append(res, getStates...), err

}

// AllDomainStates gets the states of all domains in the database.
func (db DatastoreBacked) AllDomainStates() (states []DomainState, err error) {
	return db.statesForQuery(datastore.NewQuery("DomainState"))
}

// StatesWithStatus returns the states of domains with the given status in the database.
func (db DatastoreBacked) StatesWithStatus(status PreloadStatus) (domains []DomainState, err error) {
	return db.statesForQuery(
		datastore.NewQuery("DomainState").FilterField("Status", "=", string(status)))
}

// GetIneligibleDomainStates returns the state for the given domain.
func (db DatastoreBacked) GetIneligibleDomainStates(domains []string) (states []IneligibleDomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return nil, datastoreErr
	}

	get := func(keys []*datastore.Key) ([]IneligibleDomainState, error) {
		state := make([]IneligibleDomainState, len(keys))
		if err := client.GetMulti(c, keys, state); err != nil {
			return nil, err
		}
		for i := range state {
			state[i].Name = keys[i].Name
		}
		return state, nil
	}

	var keys []*datastore.Key
	for _, domain := range domains {
		key := datastore.NameKey(ineligibleDomainStateKind, domain, nil)
		keys = append(keys, key)
		if len(keys) >= batchSize {
			if _, err := get(keys); err != nil {
				return nil, err
			}
			keys = keys[:0]
		}
	}
	return get(keys)
}

// SetIneligibleDomainStates updates the given domains updates in batches.
// Writes updates to logf in real-time.
func (db DatastoreBacked) SetIneligibleDomainStates(updates []IneligibleDomainState, logf func(format string, args ...interface{})) error {

	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return datastoreErr
	}

	set := func(keys []*datastore.Key, values []IneligibleDomainState) error {

		logf("Updating %d entries...", len(keys))

		if _, err := client.PutMulti(c, keys, values); err != nil {
			logf(" failed.\n")
			return err
		}

		logf(" done.\n")
		return nil
	}

	var keys []*datastore.Key
	var values []IneligibleDomainState
	for _, state := range updates {
		key := datastore.NameKey(ineligibleDomainStateKind, state.Name, nil)
		keys = append(keys, key)
		values = append(values, state)
		if len(keys) >= batchSize {
			if err := set(keys, values); err != nil {
				return err
			}
			keys = keys[:0]
			values = values[:0]
		}
	}
	return set(keys, values)
}

// DeleteIneligibleDomainStates deletes the state for the given domain from the database
func (db DatastoreBacked) DeleteIneligibleDomainStates(domains []string) (err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return datastoreErr
	}
	delete := func(keys []*datastore.Key) error {
		if err := client.DeleteMulti(c, keys); err != nil {
			return err
		}
		return nil
	}

	var keys []*datastore.Key
	for _, domain := range domains {
		key := datastore.NameKey(ineligibleDomainStateKind, domain, nil)
		keys = append(keys, key)
		if len(keys) >= batchSize {
			if err := delete(keys); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	return delete(keys)
}

// GetAllIneligibleDomainStates returns all the ineligible domains in the database
func (db DatastoreBacked) GetAllIneligibleDomainStates() (states []IneligibleDomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return states, datastoreErr
	}

	keys, err := client.GetAll(c, datastore.NewQuery("IneligibleDomainState"), &states)
	if err != nil {
		return states, err
	}

	for i, key := range keys {
		state := states[i]
		state.Name = key.Name
		states[i] = state
	}

	return states, nil
}

// SetPendingAutomatedRemoval sets the status of a list of domains to StatusPendingAutoamtedRemoval
func SetPendingAutomatedRemoval(db Database, domains []string, logf func(fomat string, args ...interface{})) error {
	setDomainStates := func(domainStates []DomainState) error {
		logf("Updating %d entries...", len(domainStates))

		if err := db.PutStates(domainStates, logf); err != nil {
			logf(" failed.\n")
			return err
		}
		logf(" done.\n")
		return nil
	}

	updates, err := db.StatesForDomains(domains)
	if err != nil {
		return err
	}
	for i := range updates {
		updates[i].Status = StatusPendingAutomatedRemoval
	}

	return setDomainStates(updates)
}
