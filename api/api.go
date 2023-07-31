package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromium/hstspreload.org/database"
)

// API holds the server API. Use api.New() to construct.
type API struct {
	database    database.Database
	hstspreload hstspreloadWrapper
	preloadlist preloadlistWrapper
	cache       *cache
}

const (
	defaultCacheDuration = 1 * time.Minute
)

// New creates a new API struct with the given database and the proper
// unexported fields.
func New(db database.Database) API {
	return API{
		database:    db,
		hstspreload: actualHstspreload{},
		preloadlist: actualPreloadlist{},
		cache:       cacheWithDuration(defaultCacheDuration),
	}
}

// writeJSONOrBust should only be called if nothing has been written yet.
func writeJSONOrBust(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not format JSON. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s\n", b)
}

// CheckConnection tests if we can connect the database.
func (api API) CheckConnection() error {
	// Make sure we can connect to the datastore by forcing a fetch.
	_, err := api.database.StateForDomain("garron.net")
	if err != nil {
		if strings.Contains(err.Error(), "missing project/dataset id") {
			fmt.Fprintf(os.Stderr, "Try running: make serve\n")
		}
		return err
	}

	return nil
}
