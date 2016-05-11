package database

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/chromium/hstspreload.appspot.com/database/gcd"
)

var (
	dbLock sync.Mutex
	db     Database
)

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

	db = localDatabase
	exitCode := m.Run()

	shutdown()
	os.Exit(exitCode)
}

func resetDB() {
	db.backend.(gcd.LocalBackend).Reset()
}

func TestAllDomainStatesEmptyDB(t *testing.T) {
	dbLock.Lock()
	defer dbLock.Unlock()
	resetDB()

	domains, err := db.AllDomainStates()
	if err != nil {
		t.Errorf("%s", err)
		t.FailNow()
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
	dbLock.Lock()
	defer dbLock.Unlock()
	resetDB()

	for _, tt := range putAndAllTests {

		var statuses []string
		statusReport := func(format string, args ...interface{}) {
			formatted := fmt.Sprintf(format, args...)
			statuses = append(statuses, formatted)
		}

		err := db.PutStates(
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

		domainStates, err := db.AllDomainStates()
		if err != nil {
			t.Errorf("%s", err)
			t.FailNow()
		}

		if err = matchWanted(domainStates, tt.wantStates); err != nil {
			t.Errorf("[%s] Domains do not match wanted: %s", tt.description, err)
		}

	}
}

func TestStateForDomain(t *testing.T) {
	dbLock.Lock()
	defer dbLock.Unlock()
	resetDB()

	err := db.PutState(
		DomainState{Name: "gmail.com", Status: StatusPending},
	)
	if err != nil {
		t.Errorf("cannot put state %s", err)
		return
	}

	state, err := db.StateForDomain("gmail.com")
	if err != nil {
		t.Errorf("error retrieving state: %s", err)
		return
	}
	if state.Status != StatusPending {
		t.Errorf("Wrong status: %s", state.Status)
	}

	state, err = db.StateForDomain("garron.net")
	if err != nil {
		t.Errorf("error retrieving state: %s", err)
		return
	}
	if state.Status != StatusUnknown {
		t.Errorf("Wrong status: %s", state.Status)
	}
}

// Test PutStates and AllDomainStates.
func TestDomainsWithStatus(t *testing.T) {
	dbLock.Lock()
	defer dbLock.Unlock()
	resetDB()

	domainStates, err := db.DomainsWithStatus(StatusPreloaded)
	if err != nil {
		t.Errorf("%s", err)
	}
	if len(domainStates) != 0 {
		t.Errorf("Empty database should contain no preloaded domains")
	}

	err = db.PutStates(
		[]DomainState{
			{Name: "a.com", Status: StatusPending},
			{Name: "b.com", Status: StatusPending},
			{Name: "c.com", Status: StatusRejected},
			{Name: "d.com", Status: StatusRemoved},
			{Name: "e.com", Status: StatusPending},
			{Name: "g.com", Status: StatusRejected},
			{Name: "h.com", Status: StatusPreloaded},
			{Name: "i.com", Status: StatusPreloaded},
			{Name: "j.com", Status: StatusRejected},
			{Name: "k.com", Status: StatusPending},
		},
		ignoreStatus,
	)
	if err != nil {
		t.Errorf("cannot put states %s", err)
		return
	}

	table := []struct {
		status  PreloadStatus
		domains sort.StringSlice // sorted order
	}{
		{status: StatusUnknown},
		{StatusPending, sort.StringSlice{"a.com", "b.com", "e.com", "k.com"}},
		{StatusPreloaded, sort.StringSlice{"h.com", "i.com"}},
		{StatusRejected, sort.StringSlice{"c.com", "g.com", "j.com"}},
		{StatusRemoved, sort.StringSlice{"d.com"}},
	}

	for _, tt := range table {

		domainStates, err = db.DomainsWithStatus(tt.status)
		if err != nil {
			t.Errorf("%s", err)
		}
		ss := sort.StringSlice(domainStates)
		sort.Sort(ss)
		if !reflect.DeepEqual(ss, tt.domains) {
			t.Errorf("not the list of expected domains for status %s: %#v", tt.status, ss)
		}
	}
}
