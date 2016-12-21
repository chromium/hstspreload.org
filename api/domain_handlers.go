package api

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/idna"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.org/database"
)

func getASCIIDomain(wantMethod string, w http.ResponseWriter, r *http.Request) (ascii string, ok bool) {
	if r.Method != wantMethod {
		http.Error(w, fmt.Sprintf("Wrong method. Requires %s.", wantMethod), http.StatusMethodNotAllowed)
		return "", false
	}

	unicode := r.URL.Query().Get("domain")
	if unicode == "" {
		http.Error(w, "Domain not specified.", http.StatusBadRequest)
		return "", false
	}

	ascii, err := idna.ToASCII(unicode)
	if err != nil {
		msg := fmt.Sprintf("Internal error: not convert domain to ASCII. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return "", false
	}

	return ascii, true
}

// Preloadable takes a single domain and returns if it is preloadable.
//
// Example: GET /preloadable?domain=garron.net
func (api API) Preloadable(w http.ResponseWriter, r *http.Request) {
	if cont := api.allowCORS(w, r); !cont {
		return
	}

	domain, ok := getASCIIDomain(http.MethodGet, w, r)
	if !ok {
		return
	}

	_, issues := api.hstspreload.PreloadableDomain(domain)
	writeJSONOrBust(w, issues)
}

// Removable takes a single domain and returns if it is removable.
//
// Example: GET /removable?domain=garron.net
func (api API) Removable(w http.ResponseWriter, r *http.Request) {
	domain, ok := getASCIIDomain(http.MethodGet, w, r)
	if !ok {
		return
	}

	_, issues := api.hstspreload.RemovableDomain(domain)
	writeJSONOrBust(w, issues)
}

// Status takes a single domain and returns its preload status.
//
// Example: GET /status?domain=garron.net
func (api API) Status(w http.ResponseWriter, r *http.Request) {
	if cont := api.allowCORS(w, r); !cont {
		return
	}

	domain, ok := getASCIIDomain(http.MethodGet, w, r)
	if !ok {
		return
	}

	state, err := api.database.StateForDomain(domain)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve status. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	state.Name = domain
	writeJSONOrBust(w, state)
}

// Submit takes a single domain and attempts to submit it to the
// pending queue for the HSTS preload list.
//
// Although the method is POST, we currently use a URL parameter so that
// it's easy to use in the same way as the other domain endpoints.
//
// Example: POST /status?domain=garron.net
func (api API) Submit(w http.ResponseWriter, r *http.Request) {
	domain, ok := getASCIIDomain(http.MethodPost, w, r)
	if !ok {
		return
	}

	_, issues := api.hstspreload.PreloadableDomain(domain)
	if len(issues.Errors) > 0 {
		writeJSONOrBust(w, issues)
		return
	}

	state, stateErr := api.database.StateForDomain(domain)
	if stateErr != nil {
		msg := fmt.Sprintf("Internal error: could not get current domain status. (%s)\n", stateErr)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	switch state.Status {
	case database.StatusUnknown:
		fallthrough
	case database.StatusRejected:
		fallthrough
	case database.StatusRemoved:
		putErr := api.database.PutState(database.DomainState{
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
