package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromium/hstspreload.org/database"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

func TestUpdate(t *testing.T) {
	tests := []struct {
		name string
		initialDatabaseEntries []database.DomainState
		preloadListEntries []preloadlist.Entry
		expectedDatabaseEntries []database.DomainState
	}{
		{
			"add new domains",
			nil,
			[]preloadlist.Entry{
				{
					Name: "custom-policy.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Custom,
				},
				{
					Name: "bulk-legacy.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.BulkLegacy,
				},
				{
					Name: "bulk-18-weeks.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk18Weeks,
				},
				{
					Name: "bulk-1-year.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
			[]database.DomainState{
				{
					Name: "custom-policy.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.Custom,
				},
				{
					Name: "bulk-legacy.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.BulkLegacy,
				},
				{
					Name: "bulk-18-weeks.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk18Weeks,
				},
				{
					Name: "bulk-1-year.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
		},
		{
			"update pending preloading",
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusPending,
					IncludeSubDomains: true,
				},
			},
			[]preloadlist.Entry{
				{
					Name: "preloaded.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
		},
		{
			// Test the state transition when a manually managed
			// domain gets removed from the list.
			"domain removal",
			[]database.DomainState{
				{
					Name: "preloaded-custom.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.Custom,
				},
			},
			nil,
			[]database.DomainState{
				{
					Name: "preloaded-custom.test",
					Status: database.StatusRemoved,
					IncludeSubDomains: true,
					// TODO: the policy field should get cleared for removed domains.
					Policy: preloadlist.Custom,
				},
			},
		},
		{
			"update pending removal",
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusPendingRemoval,
					IncludeSubDomains: true,
				},
			},
			nil,
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusRejected,
					// TODO: This state transition should not have this message.
					Message: "Domain was added and removed without being preloaded.",
					IncludeSubDomains: true,
				},
			},
		},
		{
			"update pending automated removal",
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusPendingAutomatedRemoval,
					IncludeSubDomains: true,
				},
			},
			nil,
			[]database.DomainState{
				{
					Name: "preloaded.test",
					// TODO: This should be StatusRejected.
					Status: database.StatusPendingAutomatedRemoval,
					IncludeSubDomains: true,
				},
			},
		},
		{
			"pending removals stay pending when on list",
			[]database.DomainState{
				{
					Name: "pending-removal.test",
					Status: database.StatusPendingRemoval,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
				{
					Name: "pending-automated-removal.test",
					Status: database.StatusPendingAutomatedRemoval,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
			[]preloadlist.Entry{
				{
					Name: "pending-removal.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
				{
					Name: "pending-automated-removal.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
			[]database.DomainState{
				{
					Name: "pending-removal.test",
					Status: database.StatusPendingRemoval,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
				{
					Name: "pending-automated-removal.test",
					// TODO: This should be StatusPendingAutomatedRemoval
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
		},
		{
			// Test that Update adds missing policy information for
			// an entry whose status is unchanged.
			"add missing policy",
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
				},
			},
			[]preloadlist.Entry{
				{
					Name: "preloaded.test",
					Mode: preloadlist.ForceHTTPS,
					IncludeSubDomains: true,
					Policy: preloadlist.Bulk1Year,
				},
			},
			[]database.DomainState{
				{
					Name: "preloaded.test",
					Status: database.StatusPreloaded,
					IncludeSubDomains: true,
					// TODO: Uncomment the following line when
					// the functionality under test is fixed.
					// Policy: preloadlist.Bulk1Year,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up test state
			api, _, _, mockPreloadlist := mockAPI(0 * time.Second)
			for _, domainState := range test.initialDatabaseEntries {
				api.database.PutState(domainState)
			}
			mockPreloadlist.list = preloadlist.PreloadList{Entries: test.preloadListEntries}

			// Make the call to Update
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}
			r, err := http.NewRequest("GET", "", nil)
			if err != nil {
				t.Fatalf("NewRequest failed: %v", err)
			}
			api.Update(w, r)
			if w.Code != 200 {
				t.Errorf("Expected HTTP status code 200, got %d", w.Code)
			}

			// Verify that the updated domain states in the database
			// match the expected state.
			gotStates, err := api.database.AllDomainStates()
			if err != nil {
				t.Fatalf("Failed to get database domain states: %v", err)
			}

			expectedStateByName := make(map[string]database.DomainState)
			for _, expectedEntry := range test.expectedDatabaseEntries {
				expectedStateByName[expectedEntry.Name] = expectedEntry
			}

			if len(gotStates) != len(test.expectedDatabaseEntries) {
				t.Errorf("Expected %d entries in the database; found %d", len(test.expectedDatabaseEntries), len(gotStates))
			}
			for _, gotState := range gotStates {
				name := gotState.Name
				wantState, ok := expectedStateByName[name]
				if !ok {
					t.Errorf("State for %q unexpectedly in database", name)
					continue
				}
				if !gotState.MatchesWanted(wantState) {
					t.Errorf("For domain %q: got %+v, want %+v", name, gotState, wantState)
				}
			}
		})
	}
}
