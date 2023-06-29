package database

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload.org/database/gcd"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

// We can share a database across tests because tests are not run
// in parallel (by default).
var testDB DatastoreBacked

func ExampleTempLocalDatabase() {
	_, shutdown, err := TempLocalDatabase()
	if err != nil {
		fmt.Printf("%s", err)
	}
	defer shutdown()
}

func TestMain(m *testing.M) {
	localDatabase, shutdown, err := TempLocalDatabase()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not initialize local backend")
		os.Exit(1)
	}

	testDB = localDatabase
	exitCode := m.Run()

	shutdown()
	os.Exit(exitCode)
}

func resetDB() {
	testDB.backend.(gcd.LocalBackend).Reset()
}

func TestAllDomainStatesEmptyDB(t *testing.T) {
	resetDB()

	domains, err := testDB.AllDomainStates()
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(domains) != 0 {
		t.Errorf("Unexpected length: %d", len(domains))
	}
}

var putAndAllTests = []struct {
	description       string
	domainStates      []DomainState
	wantStatusReports []string
	wantStates        []DomainState
}{
	{
		"one domain",
		[]DomainState{
			{Name: "gmail.com", Status: StatusPending},
		},
		[]string{"Updating 1 entries...", " done.\n"},
		[]DomainState{
			{Name: "gmail.com", Status: StatusPending},
		},
	},
	{
		"no domains",
		[]DomainState{},
		[]string{"No updates.\n"},
		[]DomainState{
			{Name: "gmail.com", Status: StatusPending},
		},
	},
	{
		"two domains",
		[]DomainState{
			{Name: "example.com", Status: StatusRejected, Message: "not enough cowbell"},
			{Name: "garron.net", Status: StatusPreloaded},
		},
		[]string{"Updating 2 entries...", " done.\n"},
		[]DomainState{
			{Name: "gmail.com", Status: StatusPending},
			{Name: "example.com", Status: StatusRejected, Message: "not enough cowbell"},
			{Name: "garron.net", Status: StatusPreloaded},
		},
	},
	{
		"new + old",
		[]DomainState{
			{Name: "gmail.com", Status: StatusUnknown},
			{Name: "wikipedia.org", Status: StatusPreloaded},
		},
		[]string{"Updating 2 entries...", " done.\n"},
		[]DomainState{
			{Name: "gmail.com", Status: StatusUnknown},
			{Name: "example.com", Status: StatusRejected, Message: "not enough cowbell"},
			{Name: "garron.net", Status: StatusPreloaded},
			{Name: "wikipedia.org", Status: StatusPreloaded},
		},
	},
}

// Test PutStates and AllDomainStates.
func TestPutAndAll(t *testing.T) {
	resetDB()

	for _, tt := range putAndAllTests {

		var statuses []string
		statusReport := func(format string, args ...interface{}) {
			formatted := fmt.Sprintf(format, args...)
			statuses = append(statuses, formatted)
		}

		err := testDB.PutStates(
			tt.domainStates,
			statusReport,
		)
		if err != nil {
			t.Errorf("[%s] cannot put states %s", tt.description, err)
			return
		}

		if !reflect.DeepEqual(statuses, tt.wantStatusReports) {
			t.Errorf("[%s] Incorrect status reports: %#v", tt.description, statuses)
		}

		domainStates, err := testDB.AllDomainStates()
		if err != nil {
			t.Fatalf("%s", err)
		}

		if !MatchWanted(domainStates, tt.wantStates) {
			t.Errorf("[%s] Domains do not match wanted: %s", tt.description, err)
		}

	}
}

func TestStateForDomain(t *testing.T) {
	resetDB()

	err := testDB.PutState(
		DomainState{Name: "gmail.com", Status: StatusPending},
	)
	if err != nil {
		t.Errorf("cannot put state %s", err)
		return
	}

	state, err := testDB.StateForDomain("gmail.com")
	if err != nil {
		t.Errorf("error retrieving state: %s", err)
		return
	}
	if state.Status != StatusPending {
		t.Errorf("Wrong status: %s", state.Status)
	}

	state, err = testDB.StateForDomain("garron.net")
	if err != nil {
		t.Errorf("error retrieving state: %s", err)
		return
	}
	if state.Status != StatusUnknown {
		t.Errorf("Wrong status: %s", state.Status)
	}
}

// Test PutStates and AllDomainStates.
func TestStatesWithStatus(t *testing.T) {
	domainA := DomainState{Name: "a.com", Status: StatusPending, IncludeSubDomains: true, Policy: preloadlist.Test}
	domainB := DomainState{Name: "b.com", Status: StatusPending, IncludeSubDomains: true, Policy: preloadlist.Custom}
	domainC := DomainState{Name: "c.com", Status: StatusRejected, IncludeSubDomains: false, Policy: preloadlist.UnspecifiedPolicyType}
	domainD := DomainState{Name: "d.com", Status: StatusRemoved, IncludeSubDomains: true, Policy: preloadlist.Bulk18Weeks}
	domainE := DomainState{Name: "e.com", Status: StatusPending, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year}
	domainG := DomainState{Name: "g.com", Status: StatusRejected, IncludeSubDomains: false, Policy: preloadlist.Test}
	domainH := DomainState{Name: "h.com", Status: StatusPreloaded, IncludeSubDomains: true, Policy: preloadlist.Custom}
	domainI := DomainState{Name: "i.com", Status: StatusPreloaded, IncludeSubDomains: false, Policy: preloadlist.UnspecifiedPolicyType}
	domainJ := DomainState{Name: "j.com", Status: StatusRejected, IncludeSubDomains: false, Policy: preloadlist.Bulk18Weeks}
	domainK := DomainState{Name: "k.com", Status: StatusPending, IncludeSubDomains: true, Policy: preloadlist.Bulk1Year}
	resetDB()

	domainStates, err := testDB.StatesWithStatus(StatusPreloaded)
	if err != nil {
		t.Errorf("%s", err)
	}
	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no preloaded domains")
	}

	err = testDB.PutStates(
		[]DomainState{
			domainA, domainB, domainC, domainD, domainE, domainG, domainH, domainI, domainJ, domainK,
		},
		blackholeLogf,
	)
	if err != nil {
		t.Errorf("cannot put states %s", err)
		return
	}

	table := []struct {
		status  PreloadStatus
		domains []DomainState // sorted order
	}{
		{status: StatusUnknown},
		{StatusPending, []DomainState{domainA, domainB, domainE, domainK}},
		{StatusPreloaded, []DomainState{domainH, domainI}},
		{StatusRejected, []DomainState{domainC, domainG, domainJ}},
		{StatusRemoved, []DomainState{domainD}},
	}

	for _, tt := range table {

		domainStates, err = testDB.StatesWithStatus(tt.status)
		if err != nil {
			t.Errorf("%s", err)
		}
		sort.Slice(domainStates, func(i, j int) bool { return domainStates[i].Name < domainStates[j].Name })
		if len(domainStates) != len(tt.domains) {
			t.Errorf("Incorrect count of states for status %s", tt.status)
		}
		for i, domainState := range domainStates {
			if !domainState.Equal(tt.domains[i]) {
				t.Errorf("unexpected domain at position %d for status %s: %#v", i, tt.status, domainState)
			}
		}
	}
}

// setIneligibleDomainTests is a struct that is used in testing SetIneligibleDomainStates
var setIneligibleDomainTests = []struct {
	description       string
	domainStates      []IneligibleDomainState
	wantStatusReports []string
}{
	{
		"two domains",
		[]IneligibleDomainState{
			{Name: "youtube.test", Policy: "bulk-1-year", Scans: []Scan{
				{
					ScanTime: time.Unix(1234, 54324),
					Issues: []hstspreload.Issues{
						{
							Errors: []hstspreload.Issue{
								{
									Code:    "formatting error",
									Summary: "missing end of page line",
									Message: "add a line at the end of the page",
								},
							},
						},
					},
				},
			}},
			{Name: "garron.test", Policy: "bulk-1-year"},
		},
		[]string{"Updating 2 entries...", " done.\n"},
	},
	{
		"bulk-18-week",
		[]IneligibleDomainState{{Name: "gmail.test", Policy: "bulk-18-week", Scans: []Scan{
			{
				ScanTime: time.Unix(1234, 54324),
				Issues: []hstspreload.Issues{
					{
						Errors: []hstspreload.Issue{
							{
								Code:    "invalid domain",
								Summary: "domain does not exist",
								Message: "domain name added does not exist",
							},
						},
					},
				},
			},
		}}},
		[]string{"Updating 1 entries...", " done.\n"},
	},
	{
		"bulk-1-year",
		[]IneligibleDomainState{{Name: "wikipedia.test", Policy: "bulk-1-year"}},
		[]string{"Updating 1 entries...", " done.\n"},
	},
}

// Test SetIneligibleDomainStates is testing adding IneligibleDomainStates
// to the database
func TestSetIneligibleDomainStates(t *testing.T) {
	resetDB()

	for _, tt := range setIneligibleDomainTests {

		var statuses []string
		statusReport := func(format string, args ...interface{}) {
			formatted := fmt.Sprintf(format, args...)
			statuses = append(statuses, formatted)
		}

		err := testDB.SetIneligibleDomainStates(
			tt.domainStates,
			statusReport,
		)
		if err != nil {
			t.Errorf("[%s] cannot put states %s", tt.description, err)
			return
		}

		if !reflect.DeepEqual(statuses, tt.wantStatusReports) {
			t.Errorf("[%s] Incorrect status reports: %#v", tt.description, statuses)
		}
	}
}

// getAndDeleteTests is a struct that is used in testing GetIneligibleDomainStates
// and DeleteIneligibleDomainStates
var getAndDeleteTests = []struct {
	description string
	domainNames []string
	wantStates  []IneligibleDomainState
}{
	{
		"one domain",
		[]string{"a.test"},
		[]IneligibleDomainState{
			{Name: "a.test", Policy: "bulk-1-year", Scans: []Scan{
				{
					ScanTime: time.Unix(1234, 54324),
					Issues: []hstspreload.Issues{
						{
							Errors: []hstspreload.Issue{
								{
									Code:    "formatting error",
									Summary: "missing end of page line",
									Message: "add a line at the end of the page",
								},
							},
						},
					},
				},
			}},
		},
	},
	{
		"two domains",
		[]string{"b.test", "c.test"},
		[]IneligibleDomainState{{Name: "b.test", Policy: "bulk-18-week", Scans: []Scan{
			{
				ScanTime: time.Unix(1234, 54324),
				Issues: []hstspreload.Issues{
					{
						Errors: []hstspreload.Issue{
							{
								Code:    "formatting error",
								Summary: "missing end of page line",
								Message: "add a line at the end of the page",
							},
						},
					},
				},
			},
		}},
			{Name: "c.test", Policy: "bulk-18-week", Scans: []Scan{
				{
					ScanTime: time.Unix(1234, 54324),
					Issues: []hstspreload.Issues{
						{
							Errors: []hstspreload.Issue{
								{
									Code:    "example error",
									Summary: "missing example",
									Message: "add example to code",
								},
							},
						},
					},
				},
			}}},
	},
	{
		"multiple domains",
		[]string{"e.test", "f.test", "g.test", "h.test", "i.test", "j.test", "k.test"},
		[]IneligibleDomainState{
			{Name: "e.test", Policy: "bulk-1-year"},
			{Name: "f.test", Policy: "bulk-1-year"},
			{Name: "g.test", Policy: "bulk-18-week"},
			{Name: "h.test", Policy: "bulk-18-week"},
			{Name: "i.test", Policy: "bulk-1-year"},
			{Name: "j.test", Policy: "bulk-18-week"},
			{Name: "k.test", Policy: "bulk-1-year"},
		},
	},
}

// Test GetIneligibleDomainStates tests getting IneligibleDomainStates from the
// database

func TestGetIneligibleDomainStates(t *testing.T) {
	resetDB()

	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}
	for _, tt := range getAndDeleteTests {

		// add domains to the database
		err := testDB.SetIneligibleDomainStates(
			tt.wantStates,
			statusReport,
		)
		if err != nil {
			t.Errorf("[%s] cannot put states %s", tt.description, err)
			return
		}

		// get domains from the database
		domainStates, err := testDB.GetIneligibleDomainStates(tt.domainNames)
		if err != nil {
			t.Errorf("%s", err)
		}

		sort.Slice(domainStates, func(i, j int) bool { return domainStates[i].Name < domainStates[j].Name })
		if len(domainStates) != len(tt.domainNames) {
			t.Errorf("Incorrect count of states for test %s", tt.description)
		}
		for i, domainState := range domainStates {
			if domainState.Name != (tt.domainNames[i]) {
				t.Errorf("unexpected domain at position %d for test %s: %#v", i, tt.description, domainState)
			}
		}
	}
}

// TestDeleteIneligibleDomainStates tests deleting IneligibleDomainStates from the database
func TestDeleteIneligibleDomainStates(t *testing.T) {
	resetDB()

	var domainStates []IneligibleDomainState

	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}
	for _, tt := range getAndDeleteTests {
		// add domains to the database
		err := testDB.SetIneligibleDomainStates(
			tt.wantStates,
			statusReport,
		)
		if err != nil {
			t.Errorf("[%s] cannot put states %s", tt.description, err)
			return
		}

		// delete domains from the database
		err = testDB.DeleteIneligibleDomainStates(tt.domainNames)
		if err != nil {
			t.Errorf("%s", err)
		}

		// get domains from the database
		// should not exist as they are deleted
		domainStates, err = testDB.GetIneligibleDomainStates(tt.domainNames)

		if merr, ok := err.(datastore.MultiError); ok {
			for _, err := range merr {
				if err != datastore.ErrNoSuchEntity {
					t.Errorf("%s", err)
				}
			}
		} else {
			if err != datastore.ErrNoSuchEntity {
				t.Errorf("%s", err)
			}
		}
	}

	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no ineligible domains")
	}

}

// TestSetDuplicateIneligibleDomainStates is testing adding duplicate
// IneligibleDomainStates to the database
func TestSetDuplicateIneligibleDomainStates(t *testing.T) {
	resetDB()

	// add duplicate domains to the database
	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}
	err := testDB.SetIneligibleDomainStates(
		[]IneligibleDomainState{{Name: "gmail.test", Policy: "bulk-18-week", Scans: []Scan{
			{
				ScanTime: time.Unix(1234, 54324),
				Issues: []hstspreload.Issues{
					{
						Errors: []hstspreload.Issue{
							{
								Code:    "format_error",
								Summary: "Formatting error",
								Message: "Please fix the format in your code",
							},
						},
					},
				},
			},
		}},
			{Name: "gmail.test", Policy: "bulk-18-week", Scans: []Scan{
				{
					ScanTime: time.Unix(1234, 54324),
					Issues: []hstspreload.Issues{
						{
							Errors: []hstspreload.Issue{
								{
									Code:    "format_error",
									Summary: "Formatting error",
									Message: "Please fix the format in your code",
								},
							},
						},
					},
				},
			}}},
		statusReport,
	)
	if err.Error() != "rpc error: code = InvalidArgument desc = A non-transactional commit may not contain multiple mutations affecting the same entity." {
		t.Errorf("[duplicate domains] cannot put states %s", err)
		return
	}
}

// TestGetDuplicateIneligibleDomainStates is testing getting duplicate
// IneligibleDomainStates from the database
func TestGetDuplicateIneligibleDomainStates(t *testing.T) {
	resetDB()

	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}

	// add domains to the database
	err := testDB.SetIneligibleDomainStates(
		[]IneligibleDomainState{{Name: "gmail.test", Policy: "bulk-18-week", Scans: []Scan{
			{
				ScanTime: time.Unix(1234, 54324),
				Issues: []hstspreload.Issues{
					{
						Errors: []hstspreload.Issue{
							{
								Code:    "invalid domain",
								Summary: "domain does not exist",
								Message: "domain name added does not exist",
							},
						},
					},
				},
			},
		}}},
		statusReport,
	)
	if err != nil {
		t.Errorf("[getIneligibleDuplicateDomain] cannot put states %s", err)
		return
	}

	// get duplicate domains from the database
	var domainNames = []string{"gmail.test", "gmail.test"}

	domainStates, err := testDB.GetIneligibleDomainStates(domainNames)
	if err != nil {
		t.Errorf("%s", err)
	}

	sort.Slice(domainStates, func(i, j int) bool { return domainStates[i].Name < domainStates[j].Name })
	if len(domainStates) != len(domainNames) {
		t.Errorf("Incorrect count of states for test getDuplicateDomains")
	}
	for i, domainState := range domainStates {
		if domainState.Name != (domainNames[i]) {
			t.Errorf("unexpected domain at position %d for test getDuplicateDomains: %#v", i, domainState)
		}
	}
}

// TestDeleteDuplicateIneligibleDomainStates is testing deleting duplicate
// IneligibleDomainStates from the database
func TestDeleteDuplicateIneligibleDomainStates(t *testing.T) {
	resetDB()

	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}
	// add domains to the database
	err := testDB.SetIneligibleDomainStates(
		[]IneligibleDomainState{{Name: "gmail.test", Policy: "bulk-18-week", Scans: []Scan{
			{
				ScanTime: time.Unix(1234, 54324),
				Issues: []hstspreload.Issues{
					{
						Errors: []hstspreload.Issue{
							{
								Code:    "invalid domain",
								Summary: "domain does not exist",
								Message: "domain name added does not exist",
							},
						},
					},
				},
			},
		}}},
		statusReport,
	)
	if err != nil {
		t.Errorf("[deleteIneligibleDuplicateDomains] cannot put states %s", err)
		return
	}

	var domainNames = []string{"gmail.test", "gmail.test"}
	err = testDB.DeleteIneligibleDomainStates(domainNames)
	if err != nil {
		t.Errorf("%s", err)
	}

	// get domains from the database
	// should not exist as they are deleted
	domainStates, err := testDB.GetIneligibleDomainStates([]string{"gmail.test"})

	if merr, ok := err.(datastore.MultiError); ok {
		for _, err := range merr {
			if err != datastore.ErrNoSuchEntity {
				t.Errorf("%s", err)
			}
		}
	} else {
		if err != datastore.ErrNoSuchEntity {
			t.Errorf("%s", err)
		}
	}

	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no ineligible domains")
	}
}
