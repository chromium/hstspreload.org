package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/idna"

	"github.com/chromium/hstspreload.appspot.com/database"
)

// API holds the server API.
type API struct {
	DatastoreBackend database.DatastoreBackend
}

// Preloadable takes a single domain and returns if it is preloadable.
//
// Example: GET /preloadable?domain=garron.net
func (api API) Preloadable(w http.ResponseWriter, r *http.Request) {
	domainHandler(api.DatastoreBackend, "GET", preloadable, w, r)
}

// Removable takes a single domain and returns if it is removable.
//
// Example: GET /removable?domain=garron.net
func (api API) Removable(w http.ResponseWriter, r *http.Request) {
	domainHandler(api.DatastoreBackend, "GET", removable, w, r)
}

// Status takes a single domain and returns its preload status.
//
// Example: GET /status?domain=garron.net
func (api API) Status(w http.ResponseWriter, r *http.Request) {
	domainHandler(api.DatastoreBackend, "GET", status, w, r)
}

// Submit takes a single domain and attempts to submit it to the
// pending queue for the HSTS preload list.
//
// Although the method is POST, we currently use a URL parameter so that
// it's easy to use in the same way as the other domain endpoints.
//
// Example: POST /status?domain=garron.net
func (api API) Submit(w http.ResponseWriter, r *http.Request) {
	domainHandler(api.DatastoreBackend, "GET", submit, w, r)
}

// Pending returns a list of domains with status "pending".
//
// Example: GET /pending
func (api API) Pending(w http.ResponseWriter, r *http.Request) {
	pending(api.DatastoreBackend, w, r)
}

// Update tells the server to update pending/removed entries based
// on the HSTS preload list source.
//
// Example: GET /pending
func (api API) Update(w http.ResponseWriter, r *http.Request) {
	update(api.DatastoreBackend, w, r)
}

func domainHandler(db database.DatastoreBackend, method string, handler func(database.DatastoreBackend, http.ResponseWriter, string), w http.ResponseWriter, r *http.Request) {
	if r.Method != method {
		http.Error(w, fmt.Sprintf("Wrong method. Requires %s.", method), http.StatusMethodNotAllowed)
		return
	}

	unicode := r.URL.Query().Get("domain")
	if unicode == "" {
		http.Error(w, "Domain not specified.", http.StatusBadRequest)
		return
	}

	ascii, err := idna.ToASCII(unicode)
	if err != nil {
		msg := fmt.Sprintf("Internal error: not convert domain to ASCII. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	handler(db, w, ascii)
}

// writeJSONOrBust should only be called if nothing has been written yet.
func writeJSONOrBust(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-type", "text/css; charset=utf-8")

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not format JSON. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s\n", b)
}

// TestConnection tests if we can connect the database.
func (api API) TestConnection() error {
	// Make sure we can connect to the datastore by forcing a fetch.
	_, err := database.StateForDomain(api.DatastoreBackend, "garron.net")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		if strings.Contains(err.Error(), "missing project/dataset id") {
			fmt.Fprintf(os.Stderr, "Try running: make serve\n")
		}
		return err
	}

	return nil
}
