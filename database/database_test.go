package database

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/chromium/hstspreload.org/database/gcd"
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
			{Name: "gmail.test", Status: StatusPending},
		},
		[]string{"Updating 1 entries...", " done.\n"},
		[]DomainState{
			{Name: "gmail.test", Status: StatusPending},
		},
	},
	{
		"no domains",
		[]DomainState{},
		[]string{"No updates.\n"},
		[]DomainState{
			{Name: "gmail.test", Status: StatusPending},
		},
	},
	{
		"two domains",
		[]DomainState{
			{Name: "example.test", Status: StatusRejected, Message: "not enough cowbell"},
			{Name: "garron.test", Status: StatusPreloaded},
		},
		[]string{"Updating 2 entries...", " done.\n"},
		[]DomainState{
			{Name: "gmail.test", Status: StatusPending},
			{Name: "example.test", Status: StatusRejected, Message: "not enough cowbell"},
			{Name: "garron.test", Status: StatusPreloaded},
		},
	},
	{
		"new + old",
		[]DomainState{
			{Name: "gmail.test", Status: StatusUnknown},
			{Name: "wikipedia.test", Status: StatusPreloaded},
		},
		[]string{"Updating 2 entries...", " done.\n"},
		[]DomainState{
			{Name: "gmail.test", Status: StatusUnknown},
			{Name: "example.test", Status: StatusRejected, Message: "not enough cowbell"},
			{Name: "garron.test", Status: StatusPreloaded},
			{Name: "wikipedia.test", Status: StatusPreloaded},
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
		DomainState{Name: "gmail.test", Status: StatusPending},
	)
	if err != nil {
		t.Errorf("cannot put state %s", err)
		return
	}

	state, err := testDB.StateForDomain("gmail.test")
	if err != nil {
		t.Errorf("error retrieving state: %s", err)
		return
	}
	if state.Status != StatusPending {
		t.Errorf("Wrong status: %s", state.Status)
	}

	state, err = testDB.StateForDomain("garron.test")
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
	domainA := DomainState{Name: "a.test", Status: StatusPending, IncludeSubDomains: true}
	domainB := DomainState{Name: "b.test", Status: StatusPending, IncludeSubDomains: true}
	domainC := DomainState{Name: "c.test", Status: StatusRejected, IncludeSubDomains: false}
	domainD := DomainState{Name: "d.test", Status: StatusRemoved, IncludeSubDomains: true}
	domainE := DomainState{Name: "e.test", Status: StatusPending, IncludeSubDomains: true}
	domainG := DomainState{Name: "g.test", Status: StatusRejected, IncludeSubDomains: false}
	domainH := DomainState{Name: "h.test", Status: StatusPreloaded, IncludeSubDomains: true}
	domainI := DomainState{Name: "i.test", Status: StatusPreloaded, IncludeSubDomains: false}
	domainJ := DomainState{Name: "j.test", Status: StatusRejected, IncludeSubDomains: false}
	domainK := DomainState{Name: "k.test", Status: StatusPending, IncludeSubDomains: true}
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

// setTests is a struct that is used in testing SetIneligibleDomainStates
var setTests = []struct {
	description       string
	domainStates      []IneligibleDomainState
	wantStatusReports []string
}{
	{
		"two domains",
		[]IneligibleDomainState{
			{Name: "youtube.test", Policy: "bulk-1-year"},
			{Name: "garron.test", Policy: "bulk-1-year"},
		},
		[]string{"Updating 2 entries...", " done.\n"},
	},
	{
		"bulk-18-week",
		[]IneligibleDomainState{{Name: "gmail.test", Policy: "bulk-18-week"}},
		[]string{"Updating 1 entries...", " done.\n"},
	},
	{
		"bulk-1-year",
		[]IneligibleDomainState{{Name: "wikipedia.test", Policy: "bulk-1-year"}},
		[]string{"Updating 1 entries...", " done.\n"},
	},
}

// Test SetIneligibleDomainStates
func TestSetIneligibleDomainStates(t *testing.T) {
	resetDB()

	for _, tt := range setTests {

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
	description  string
	domainStates []string
	wantStates   []IneligibleDomainState
}{
	{
		"one domain",
		[]string{"a.test"},
		[]IneligibleDomainState{
			{Name: "a.test", Policy: "bulk-1-year"},
		},
	},
	{
		"two domains",
		[]string{"b.test", "c.test"},
		[]IneligibleDomainState{{Name: "b.test", Policy: "bulk-18-week"},
			{Name: "c.test", Policy: "bulk-18-week"}},
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

// Test GetIneligibleDomainStates
func TestGetIneligibleDomainStates(t *testing.T) {

	resetDB()

	// domainStates should be empty as domains are not added to database
	domainStates, err := testDB.GetIneligibleDomainStates([]string{"a.test"})
	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no preloaded domains")
	}

	// add domains to the database
	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}
	for _, tt := range getAndDeleteTests {

		err := testDB.SetIneligibleDomainStates(
			tt.wantStates,
			statusReport,
		)
		if err != nil {
			t.Errorf("[%s] cannot put states %s", tt.description, err)
			return
		}
	}
	// get domains from the database
	for _, tr := range getAndDeleteTests {
		domainStates, err = testDB.GetIneligibleDomainStates(tr.domainStates)
		if err != nil {
			t.Errorf("%s", err)
		}

		sort.Slice(domainStates, func(i, j int) bool { return domainStates[i].Name < domainStates[j].Name })
		if len(domainStates) != len(tr.domainStates) {
			t.Errorf("Incorrect count of states for test %s", tr.description)
		}
		for i, domainState := range domainStates {
			if domainState.Name != (tr.domainStates[i]) {
				t.Errorf("unexpected domain at position %d for test %s: %#v", i, tr.description, domainState)
			}
		}
	}
}

// TestDeleteIneligibleDomainStates tests the DeleteIneligibleDomainStates function
func TestDeleteIneligibleDomainStates(t *testing.T) {

	resetDB()

	// domainStates should be empty as domains are not added to database
	domainStates, err := testDB.GetIneligibleDomainStates([]string{"a.test"})

	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no preloaded domains")
	}

	// add domains to the database
	var statuses []string
	statusReport := func(format string, args ...interface{}) {
		formatted := fmt.Sprintf(format, args...)
		statuses = append(statuses, formatted)
	}
	for _, tt := range getAndDeleteTests {
		err := testDB.SetIneligibleDomainStates(
			tt.wantStates,
			statusReport,
		)
		if err != nil {
			t.Errorf("[%s] cannot put states %s", tt.description, err)
			return
		}
	}
	// delete domains from the database
	for _, tr := range getAndDeleteTests {
		err = testDB.DeleteIneligibleDomainStates(tr.domainStates)
		if err != nil {
			t.Errorf("%s", err)
		}
	}

	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no preloaded domains")
	}

}
