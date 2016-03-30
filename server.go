package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload/chromiumpreload"
)

func main() {
	staticHandler := http.FileServer(http.Dir("static"))
	http.Handle("/", staticHandler)
	http.Handle("/style.css", staticHandler)
	http.Handle("/index.js", staticHandler)

	http.HandleFunc("/robots.txt", http.NotFound)
	http.HandleFunc("/favicon.ico", http.NotFound)

	http.HandleFunc("/checkdomain/", checkdomain)
	http.HandleFunc("/status/", status)

	http.HandleFunc("/submit/", submit)
	http.HandleFunc("/pending", pending)
	http.HandleFunc("/update", update)

	http.ListenAndServe(":8080", nil)
}

func checkdomain(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[len("/checkdomain/"):]

	issues := hstspreload.CheckDomain(domain)

	b, err := json.MarshalIndent(hstspreload.MakeSlices(issues), "", "  ")
	if err != nil {
		http.Error(w, "Internal error: could not encode JSON.", 500)
	} else {
		fmt.Fprintf(w, "%s\n", b)
	}
}

// writeJSON  should only be called if nothing has been written yet.
func writeJSONOrBust(w http.ResponseWriter, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not format JSON. (%s)\n", err)
		http.Error(w, msg, 500)
		return
	}

	fmt.Fprintf(w, "%s\n", b)
	return
}

func status(w http.ResponseWriter, r *http.Request) {
	domain := chromiumpreload.Domain(r.URL.Path[len("/status/"):])

	state, err := stateForDomain(domain)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve status. (%s)\n", err)
		http.Error(w, msg, 500)
		return
	}

	domainStateJSON := struct {
		Name    chromiumpreload.Domain `json:"name"`
		Status  string                 `json:"status,name"`
		Message string                 `json:"messsage,omitempty"`
	}{
		Name:    domain,
		Status:  statusToString[state.Status],
		Message: state.Message,
	}

	writeJSONOrBust(w, domainStateJSON)
}

func submit(w http.ResponseWriter, r *http.Request) {
	domainStr := r.URL.Path[len("/submit/"):]
	domain := chromiumpreload.Domain(domainStr)

	issues := hstspreload.CheckDomain(domainStr)
	if len(issues.Errors) > 0 {
		writeJSONOrBust(w, issues)
		return
	}

	state, stateErr := stateForDomain(domain)
	if stateErr != nil {
		msg := fmt.Sprintf("Internal error: could not get current domain status. (%s)\n", stateErr)
		http.Error(w, msg, 500)
	}

	switch state.Status {
	case StatusUnknown:
		fallthrough
	case StatusRemoved:
		putErr := putState(DomainState{
			Name:   domain,
			Status: StatusPending,
		})
		if putErr != nil {
			issues = hstspreload.Issues{
				Errors:   append(issues.Errors, "Internal error: Unable to save to the pending list."),
				Warnings: issues.Warnings,
			}
		}
	case StatusPending:
		issues = hstspreload.Issues{
			Errors:   issues.Errors,
			Warnings: append(issues.Warnings, "Domain is already pending."),
		}
	case StatusPreloaded:
		issues = hstspreload.Issues{
			Errors:   append(issues.Errors, "Domain is already preloaded."),
			Warnings: issues.Warnings,
		}
	case StatusRejected:
		rejectedMsg := fmt.Sprintf("Domain has been rejected. (%s)", state.Message)
		issues = hstspreload.Issues{
			Errors:   append(issues.Warnings, rejectedMsg),
			Warnings: issues.Warnings,
		}
	default:
		issues = hstspreload.Issues{
			Errors:   append(issues.Warnings, "Cannot preload."),
			Warnings: issues.Warnings,
		}
	}

	writeJSONOrBust(w, hstspreload.MakeSlices(issues))
}

func pending(w http.ResponseWriter, r *http.Request) {
	names, err := domainsWithStatus(StatusPending)
	if err != nil {
		msg := fmt.Sprintf("Internal error: could not retrieve pending list. (%s)\n", err)
		http.Error(w, msg, 500)
		return
	}

	fmt.Fprintf(w, "[\n")
	for i, name := range names {
		comma := ","
		if i+1 == len(names) {
			comma = ""
		}

		fmt.Fprintf(w, `    { "name": "%s", "include_subdomains": true, "mode": "force-https" }%s
`, name, comma)
	}
	fmt.Fprintf(w, "]\n")
}

func difference(from []chromiumpreload.Domain, take []chromiumpreload.Domain) (diff []chromiumpreload.Domain) {
	takeSet := make(map[chromiumpreload.Domain]bool)
	for _, elem := range take {
		takeSet[elem] = true
	}

	for _, elem := range from {
		if !takeSet[elem] {
			diff = append(diff, elem)
		}
	}

	return diff
}

func update(w http.ResponseWriter, r *http.Request) {
	// Get preload list.
	preloadList, listErr := chromiumpreload.GetLatest()
	if listErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve latest preload list. (%s)\b",
			listErr,
		)
		http.Error(w, msg, 500)
		return
	}
	var actualPreload []chromiumpreload.Domain
	for _, entry := range preloadList.Entries {
		if entry.Mode == chromiumpreload.ForceHTTPS {
			actualPreload = append(actualPreload, entry.Name)
		}
	}

	// Get domains currently recorded as preloaded.
	databasePreload, dbErr := domainsWithStatus(StatusPreloaded)
	if dbErr != nil {
		msg := fmt.Sprintf(
			"Internal error: could not retrieve domain names previously marked as preloaded. (%s)\n",
			dbErr,
		)
		http.Error(w, msg, 500)
		return
	}

	// Calculate values that are out of date.
	var updates []DomainState

	added := difference(actualPreload, databasePreload)
	for _, name := range added {
		updates = append(updates, DomainState{
			Name:   name,
			Status: StatusPreloaded,
		})
	}

	removed := difference(databasePreload, actualPreload)
	for _, name := range removed {
		updates = append(updates, DomainState{
			Name:   name,
			Status: StatusRemoved,
		})
	}

	fmt.Fprintf(w, `The preload list has %d entries.
- # of preloaded HSTS entries: %d
- # to be added in this update: %d
- # to be removed this update: %d
`,
		len(preloadList.Entries),
		len(actualPreload),
		len(added),
		len(removed),
	)

	// Create statusReport function to show progress.
	written := false
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Internal error: Could not create `http.Flusher`.", 500)
		return
	}
	statusReport := func(format string, args ...interface{}) {
		fmt.Fprintf(w, format, args...)
		f.Flush()
		written = true
	}

	// Update the database
	putErr := putStates(updates, statusReport)
	if putErr != nil {
		msg := fmt.Sprintf(
			"Internal error: datastore update failed. (%s)\n",
			putErr,
		)
		if written {
			// The header has already been sent, so we can't return 500.
			fmt.Fprintf(w, msg)
		} else {
			http.Error(w, msg, 500)
		}
		return
	}

	fmt.Fprintf(w, "Success. %d domain states updated.\n", len(updates))
}
