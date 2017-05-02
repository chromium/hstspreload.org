package api

import (
	"net/http"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.org/database"
)

// DebugSetPreloaded allows preloading a domain without any checks.
// This should only be exposed for test servers.
func (api API) DebugSetPreloaded(w http.ResponseWriter, r *http.Request) {
	domain, ok := getASCIIDomain(http.MethodPost, w, r)
	if !ok {
		return
	}

	issues := hstspreload.Issues{}

	putErr := api.database.PutState(database.DomainState{
		Name:   domain,
		Status: database.StatusPreloaded,
	})

	if putErr != nil {
		issue := hstspreload.Issue{
			Code:    "internal.server.remove.set_preloaded_failed",
			Summary: "Internal error",
			Message: "Unable to save to set as preloaded.",
		}
		issues = hstspreload.Issues{
			Errors:   append(issues.Errors, issue),
			Warnings: issues.Warnings,
		}
	}

	writeJSONOrBust(w, issues)
}
