package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/idna"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

// DomainStateWithBulk is a DomainState that also includes information about the bulk status of the domain.
type DomainStateWithBulk struct {
	*database.DomainState
	Bulk            bool   `json:"bulk"`
	PreloadedDomain string `json:"preloadedDomain"`
}

func normalizeDomain(unicode string) (string, error) {
	ascii, err := idna.ToASCII(unicode)
	if err != nil {
		return "", err
	}

	return strings.ToLower(ascii), nil
}

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

	normalized, err := normalizeDomain(unicode)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not convert domain to ASCII. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return "", false
	}

	return normalized, true
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

	bulkState, err := api.statusForDomain(domain)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve status. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if bulkState.Status == database.StatusPreloaded && bulkState.PreloadedDomain != domain {
		ancestorDomain := bulkState.PreloadedDomain
		_, ancestorHasParent := parentDomain(ancestorDomain)
		var issue hstspreload.Issue
		if ancestorHasParent {
			issue = hstspreload.Issue{
				Code:    "server.removable.subdomain",
				Summary: "Domain is subdomain of preloaded domain",
				Message: fmt.Sprintf("This domain is a subdomain of %s, which is on the preload list. To remove the HSTS policy for %s, the domain %s would need to be removed from the preload list.", ancestorDomain, domain, ancestorDomain),
			}
		} else {
			issue = hstspreload.Issue{
				Code:    "server.removable.preloaded_tld",
				Summary: "Domain is registered under a preloaded TLD",
				Message: fmt.Sprintf("The entire TLD %s is preloaded for HSTS and individual domain names cannot be removed.", ancestorDomain),
			}
		}
		issues := hstspreload.Issues{
			Errors: []hstspreload.Issue{issue},
		}
		writeJSONOrBust(w, issues)
		return
	}

	_, issues := api.hstspreload.RemovableDomain(domain)

	if bulkState.DomainState.IsProtected() {
		issue := hstspreload.Issue{
			Code:    "server.removable.protected",
			Summary: "Domain protected",
			Message: "This domain is currently protected against removal through the hstspreload.org site. Please contact us via email if you want to remove it from the preload list.",
		}
		issues = hstspreload.Issues{
			Errors:   append(issues.Errors, issue),
			Warnings: issues.Warnings,
		}
	}

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

	bulkState, err := api.statusForDomain(domain)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve status. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	writeJSONOrBust(w, bulkState)
}

func (api API) statusForDomain(domain string) (*DomainStateWithBulk, error) {
	preloadedDomain := domain
	state, err := api.stateForDomainCached(domain)
	if err != nil {
		return nil, err
	}

	if state.Status == database.StatusUnknown {
		// walk up the domain name chain.
		for ancestorDomain, ok := parentDomain(domain); ok; ancestorDomain, ok = parentDomain(ancestorDomain) {
			if ancestorState, err := api.stateForDomainCached(ancestorDomain); err == nil {
				// if an ancestor domain is preloaded and includes subdomains, set current domain status
				// to preloaded as well.
				if ancestorState.Status == database.StatusPreloaded && ancestorState.IncludeSubDomains {
					state.Status = database.StatusPreloaded
					preloadedDomain = ancestorDomain
					break
				}
			}
		}
	}

	state.Name = domain
	bulkState := &DomainStateWithBulk{
		DomainState: &state,
		Bulk:        state.IsBulk(),
	}
	if state.Status == database.StatusPreloaded {
		bulkState.PreloadedDomain = preloadedDomain
	}
	return bulkState, nil
}

// parentDomain finds the parent (immediate ancestor) domain of the input domain.
func parentDomain(domain string) (string, bool) {
	dot := strings.Index(domain, ".")
	if dot == -1 || dot == len(domain) {
		return "", false
	}
	return domain[dot+1:], true
}

// Submit takes a single domain and attempts to submit it to the
// pending queue for the HSTS preload list.
//
// Although the method is POST, we currently use a URL parameter so that
// it's easy to use in the same way as the other domain endpoints.
//
// Example: POST /submit?domain=garron.net
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
	case database.StatusPendingRemoval:
		fallthrough
	case database.StatusRemoved:
		putErr := api.database.PutState(database.DomainState{
			Name:              domain,
			Status:            database.StatusPending,
			IncludeSubDomains: true,
			SubmissionDate:    time.Now(),
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

// Remove takes a single domain and attempts to submit it to the
// removal queue for the HSTS preload list.
//
// Although the method is POST, we currently use a URL parameter so that
// it's easy to use in the same way as the other domain endpoints.
//
// Example: POST /remove?domain=garron.net
func (api API) Remove(w http.ResponseWriter, r *http.Request) {
	domain, ok := getASCIIDomain(http.MethodPost, w, r)
	if !ok {
		return
	}

	_, issues := api.hstspreload.RemovableDomain(domain)
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
		issue := hstspreload.Issue{
			Code:    "server.remove.not_preloaded",
			Summary: "Not preloaded",
			Message: "The domain is not part of the preload list, so it cannot be removed.",
		}
		issues = hstspreload.Issues{
			Errors:   issues.Errors,
			Warnings: append(issues.Warnings, issue),
		}
	case database.StatusPendingRemoval:
		issue := hstspreload.Issue{
			Code:    "server.remove.already_pending_removal",
			Summary: "Already Pending Removal",
			Message: "Domain is already pending removal.",
		}
		issues = hstspreload.Issues{
			Errors:   issues.Errors,
			Warnings: append(issues.Warnings, issue),
		}
	case database.StatusRemoved:
		issue := hstspreload.Issue{
			Code:    "server.remove.already_removed",
			Summary: "Already Removed",
			Message: "Domain has already been removed.",
		}
		issues = hstspreload.Issues{
			Errors:   issues.Errors,
			Warnings: append(issues.Warnings, issue),
		}
	case database.StatusPending:
		fallthrough
	case database.StatusPreloaded:
		if state.IsProtected() {
			issue := hstspreload.Issue{
				Code:    "server.remove.protected",
				Summary: "Domain protected",
				Message: "This domain is currently protected against removal through the hstspreload.org site. Please contact us via email if you want to remove it from the preload list.",
			}
			issues = hstspreload.Issues{
				Errors:   append(issues.Errors, issue),
				Warnings: issues.Warnings,
			}
			break
		}

		putErr := api.database.PutState(database.DomainState{
			Name:              domain,
			Status:            database.StatusPendingRemoval,
			IncludeSubDomains: false,
			SubmissionDate:    time.Now(),
		})
		if putErr != nil {
			issue := hstspreload.Issue{
				Code:    "internal.server.remove.removal_failed",
				Summary: "Internal error",
				Message: "Unable to remove from the preload list.",
			}
			issues = hstspreload.Issues{
				Errors:   append(issues.Errors, issue),
				Warnings: issues.Warnings,
			}
		}
	default:
		issue := hstspreload.Issue{
			Code:    "internal.server.remove.unknown_status",
			Summary: "Internal error",
			Message: "Cannot remove; could not find domain status.",
		}
		issues = hstspreload.Issues{
			Errors:   append(issues.Warnings, issue),
			Warnings: issues.Warnings,
		}
	}

	writeJSONOrBust(w, issues)
}

// RemoveIneligibleDomains runs eligibility checks on domains present in the
// database and change the status to PendingAutomatedRemoval if the domain
// does not follow the requirements for more than 2 crawls

// Example: GET /removeineligibledomains?
func (api API) RemoveIneligibleDomains(w http.ResponseWriter, r *http.Request) {

	// map with domain domain names and their states of domains with valid policyTypes
	policyStates := make(map[string]database.DomainState)
	// map with the names of all the domains with valid policyTypes
	var policyDomains []string
	// all domains that need to be added to the ineligible
	// domain database
	var ineligibleDomains []database.IneligibleDomainState
	// all domains that need to be deleted from the
	// ineligible domain database
	var deleteEligibleDomains []string

	// Get all domains
	domains, err := api.database.AllDomainStates()
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Filter Domains
	for _, d := range domains {
		if d.Policy == preloadlist.Bulk18Weeks || d.Policy == preloadlist.Bulk1Year {
			policyDomains = append(policyDomains, d.Name)
			policyStates[d.Name] = d
		}
	}

	// call GetIneligibleDomainStates, add to map
	states := make(map[string]database.IneligibleDomainState)
	state, err := api.database.GetIneligibleDomainStates(policyDomains)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not get domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// delete domains that exist in the ineligible database but not
	// on the preload list
	for _, s := range state {
		states[s.Name] = s
		if _, ok := policyStates[s.Name]; !ok {
			deleteEligibleDomains = append(deleteEligibleDomains, s.Name)
		}
	}

	// Store ineligible domains in slice
	for _, d := range policyStates {
		//#TODO: parallelize all the calls to EligibleDomains
		_, issues := api.hstspreload.EligibleDomain(d.Name, d.Policy)

		scan := database.Scan{
			ScanTime: time.Now(),
			Issues:   issues,
		}

		val, ok := states[d.Name]
		if len(issues.Errors) > 0 {
			if ok {
				val.Scans = append(val.Scans, scan)
				ineligibleDomains = append(ineligibleDomains, val)
			} else {
				ineligibleDomains = append(ineligibleDomains, database.IneligibleDomainState{
					Name:   d.Name,
					Scans:  []database.Scan{scan},
					Policy: string(d.Policy),
				})
			}
		}
	}

	// Delete eligible domains from the database
	err = api.database.DeleteIneligibleDomainStates(deleteEligibleDomains)

	if err != nil {
		msg := fmt.Sprintf("Internal error: could not delete domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Add ineligible domains to the database
	err = api.database.SetIneligibleDomainStates(ineligibleDomains, func(format string, args ...interface{}) {})
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Set domain status to StatusPendingAutomatedRemoval

	// Anonymous function that checks if a domain's
	// status should be changed to StatusPendingAutomatedRemoval
	scans := func(state database.IneligibleDomainState) bool {
		if len(state.Scans) < 2 {
			return false
			// duration between scans should be greater than 30 days
		} else if state.Scans[len(state.Scans)-1].ScanTime.Sub(state.Scans[0].ScanTime) > time.Duration(time.Hour)*24*30 {
			return true
		}
		return false
	}

	// Get list of names of all domains that need their status changed
	var pendingRemoval []string
	allStates, err := api.database.GetAllIneligibleDomainStates()
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not get all ineligible domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	for _, id := range allStates {
		if scans(id) {
			pendingRemoval = append(pendingRemoval, id.Name)
		}
	}

	// Change status of the domain
	domainStates, err := api.database.StatesForDomains(pendingRemoval)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not get domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	for i := range domainStates {
		domainStates[i].Status = database.StatusPendingAutomatedRemoval
	}

	// Update state in database
	err = api.database.PutStates(domainStates, func(format string, args ...interface{}) {})
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not put domains. (%s)\n", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
