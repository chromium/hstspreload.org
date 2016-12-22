package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/batch"
)

// ScanAll scans all (preloaded, pending, or formerly submitted) domains for
// preload requirements.
func (api API) ScanAll(httpWriter http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(httpWriter, "Getting domains...\n")
	domains, err := api.database.DomainsWithStatus(database.StatusPending)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve list of domains. (%s)\n", err)
		http.Error(httpWriter, msg, http.StatusInternalServerError)
		return
	}

	var objectName = objectNameForCurrentTime()

	fmt.Fprintf(httpWriter, "Creating storage object...\n")
	objWriter, cancel, err := database.ObjectWriter(objectName)
	defer cancel()
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not create object writer. (%s)\n", err)
		http.Error(httpWriter, msg, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(httpWriter, "Kicking off scan: %s\n", objectName)
	batch.Fprint(objWriter, domains)
	objWriter.Close()
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not close object writer. (%s)\n", err)
		http.Error(httpWriter, msg, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(httpWriter, " done.")
}

func objectNameForCurrentTime() string {
	t := time.Now()
	var date = t.Format("2006-01-02")
	return fmt.Sprintf("scans/%d-%s.json", t.Unix(), date)
}
