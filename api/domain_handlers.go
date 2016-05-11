package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.appspot.com/database"
)

func preloadable(db database.Database, w http.ResponseWriter, domain string) {
	_, issues := hstspreload.PreloadableDomain(domain)
	writeJSONOrBust(w, issues)
}

func removable(db database.Database, w http.ResponseWriter, domain string) {
	_, issues := hstspreload.RemovableDomain(domain)
	writeJSONOrBust(w, issues)
}

func status(db database.Database, w http.ResponseWriter, domain string) {
	state, err := db.StateForDomain(domain)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve status. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	state.Name = domain
	writeJSONOrBust(w, state)
}

func submit(db database.Database, w http.ResponseWriter, domain string) {
	_, issues := hstspreload.PreloadableDomain(domain)
	if len(issues.Errors) > 0 {
		writeJSONOrBust(w, issues)
		return
	}

	state, stateErr := db.StateForDomain(domain)
	if stateErr != nil {
		msg := fmt.Sprintf("Internal error: could not get current domain status. (%s)\n", stateErr)
		http.Error(w, msg, http.StatusInternalServerError)
	}

	switch state.Status {
	case database.StatusUnknown:
		fallthrough
	case database.StatusRejected:
		fallthrough
	case database.StatusRemoved:
		putErr := db.PutState(database.DomainState{
			Name:           domain,
			Status:         database.StatusPending,
			SubmissionDate: time.Now(),
		})
		if putErr != nil {
			issue := hstspreload.Issue{
				Code:    "internal.server.preload.save_failed",
				Summary: "Internal error",
				Message: "Unable to save to the pending list.",
			}
			issues = hstspreload.Issues{
				Errors:   append(issues.Errors, issue),
				Warnings: issues.Warnings,
			}
		}
	case database.StatusPending:
		formattedDate := state.SubmissionDate.Format("Monday, _2 January 2006")
		issue := hstspreload.Issue{
			Code:    "server.preload.already_pending",
			Summary: "Domain has already been submitted",
			Message: fmt.Sprintf("Domain is already pending. It was submitted on %s.", formattedDate),
		}
		issues = hstspreload.Issues{
			Errors:   issues.Errors,
			Warnings: append(issues.Warnings, issue),
		}
	case database.StatusPreloaded:
		issue := hstspreload.Issue{
			Code:    "server.preload.already_preloaded",
			Summary: "Domain is already preloaded",
			Message: "The domain is already preloaded.",
		}
		issues = hstspreload.Issues{
			Errors:   append(issues.Errors, issue),
			Warnings: issues.Warnings,
		}
	default:
		issue := hstspreload.Issue{
			Code:    "internal.server.preload.unknown_status",
			Summary: "Internal error",
			Message: "Cannot preload; could not find domain status.",
		}
		issues = hstspreload.Issues{
			Errors:   append(issues.Warnings, issue),
			Warnings: issues.Warnings,
		}
	}

	writeJSONOrBust(w, issues)
}
