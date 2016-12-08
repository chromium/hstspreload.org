package api

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/chromium/hstspreload.appspot.com/database"
)

// TestCron is for testing cron on Google Cloud Flexible Environments:
// https://cloud.google.com/appengine/docs/flexible/go/scheduling-jobs-with-cron-yaml
func (api API) TestCron(w http.ResponseWriter, r *http.Request) {
	// TOD: Check X-Appengine-Cron
	t := time.Now()
	api.database.PutState(database.DomainState{
		Name:           "cron.test",
		Status:         database.StatusRejected,
		SubmissionDate: time.Now(),
		Message:        fmt.Sprintf("Last updated: %s (Go version: %s)", t, runtime.Version()),
	})
	http.Error(w, fmt.Sprintf("Cron ran: %s (Go version: %s)", t, runtime.Version()), http.StatusAccepted)
}
