package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/chromium/hstspreload.appspot.com/database"
)

// API holds the server API.
type API struct {
	Database database.Database
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

// CheckConnection tests if we can connect the database.
func (api API) CheckConnection() error {
	// Make sure we can connect to the datastore by forcing a fetch.
	_, err := api.Database.StateForDomain("garron.net")
	if err != nil {
		if strings.Contains(err.Error(), "missing project/dataset id") {
			fmt.Fprintf(os.Stderr, "Try running: make serve\n")
		}
		return err
	}

	return nil
}
