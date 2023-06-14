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
	timeout   = 45 * time.Second

	domainStateKind        = "DomainState"
	invalidDomainStateKind = "InvalidDomainState"
)

// A Database is an abstraction over Datastore with hstspreload-specific
// database functions.
type Database interface {
	PutStates([]DomainState, func(string, ...interface{})) error
	PutState(DomainState) error
	StateForDomain(string) (DomainState, error)
	AllDomainStates() ([]DomainState, error)
	StatesWithStatus(PreloadStatus) ([]DomainState, error)
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

// domainsForQuery returns the domains that match the given datastore query.
func (db DatastoreBacked) domainsForQuery(query *datastore.Query) (domains []string, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return domains, datastoreErr
	}

	keys, err := client.GetAll(c, query.KeysOnly(), nil)
	if err != nil {
		return domains, err
	}

	for _, key := range keys {
		domain := key.Name
		domains = append(domains, domain)
	}

	return domains, nil
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

// AllDomainStates gets the states of all domains in the database.
func (db DatastoreBacked) AllDomainStates() (states []DomainState, err error) {
	return db.statesForQuery(datastore.NewQuery("DomainState"))
}

// StatesWithStatus returns the states of domains with the given status in the database.
func (db DatastoreBacked) StatesWithStatus(status PreloadStatus) (domains []DomainState, err error) {
	return db.statesForQuery(
		datastore.NewQuery("DomainState").Filter("Status =", string(status)))
}

type Datastore interface {
	GetInvalidDomains(string) (InvalidDomainState, error)
	SetInvalidDomains([]InvalidDomainState, func(string, ...interface{}))
	DeleteInvalidDomains([]InvalidDomainState) error
}

// GetInvalidDomain returns the state for the given domain.
func (db DatastoreBacked) GetInvalidDomains(domain []string) (state []InvalidDomainState, err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return state, datastoreErr
	}
	for _, status := range domain{ // change to for each after testing
		key := datastore.NameKey(invalidDomainStateKind, status, nil)
		states := InvalidDomainState{}
		getErr := client.Get(c, key, &states)
		if getErr != nil {
			if getErr == datastore.ErrNoSuchEntity {
				println("Does not exist")
				return state, nil
			}
			return state, getErr
		}
		states.Name = key.Name
		state = append(state, states)
	}
	return state, nil
}

// SetInvalidDomains updates the given domains updates in batches.
// Writes updates to logf in real-time.
func (db DatastoreBacked) SetInvalidDomains(updates []InvalidDomainState, logf func(format string, args ...interface{})) error {

	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return datastoreErr
	}

	set := func(keys []*datastore.Key, values []InvalidDomainState) error {
		
		logf("Updating %d entries...", len(keys))

		if _, err := client.PutMulti(c, keys, values); err != nil {
			logf(" failed.\n")
			return err
		}

		logf(" done.\n")
		return nil
	}

	var keys []*datastore.Key
	var values []InvalidDomainState
	for _, state := range updates {
		key := datastore.NameKey(invalidDomainStateKind, state.Name, nil)
		keys = append(keys, key)
		values = append(values, state)
		for p, value := range values {
			println(p, ": values: ", value.Name)
		}
		if len(keys) >= batchSize {
			if err := set(keys, values); err != nil {
				println("error here")
				return err
			}
			keys = keys[:0]
			values = values[:0]
		}
	}

	return set(keys, values)
}

// DeleteInvalidDomain deletes the state for the given domain from the datbase
func (db DatastoreBacked) DeleteInvalidDomains(domain []string) (err error) {
	// Set up the datastore context.
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, datastoreErr := db.backend.NewClient(c, db.projectID)
	if datastoreErr != nil {
		return datastoreErr
	}
	for _, state := range domain{ 
		key := datastore.NameKey(invalidDomainStateKind, state, nil)
		getErr := client.Delete(c, key)
		if getErr != nil {
			if getErr == datastore.ErrNoSuchEntity {
				return nil
			}
			return getErr
		}
	}
	return nil
}