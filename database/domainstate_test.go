package database

import (
	"fmt"
	"testing"
)

func getDomain(states []DomainState, domain string) (DomainState, error) {
	for _, s := range states {
		if s.Name == domain {
			return s, nil
		}
	}
	return DomainState{}, fmt.Errorf("could not find domain state")
}

func matchesWanted(actual DomainState, wanted DomainState) bool {
	if wanted.Name != actual.Name {
		return false
	}
	if wanted.Status != actual.Status {
		return false
	}
	if wanted.Message != "" && wanted.Message != actual.Message {
		return false
	}
	return true
}

func matchWanted(actual []DomainState, wanted []DomainState) error {
	m := make(map[string]bool)
	for _, ws := range wanted {
		if m[ws.Name] {
			return fmt.Errorf("repeated wanted domain: %s", ws.Name)
		}
		m[ws.Name] = true
	}

	if len(actual) != len(wanted) {
		return fmt.Errorf(
			"number of states (%d) do not match expected (%d)",
			len(actual),
			len(wanted),
		)
	}

	for _, ws := range wanted {
		s, err := getDomain(actual, ws.Name)
		if err != nil {
			return fmt.Errorf("domain %s not present", ws.Name)
		}
		if !matchesWanted(s, ws) {
			return fmt.Errorf("does not match wanted for domain %s: %#v", ws.Name, s)
		}
	}

	return nil
}

func TestMatchWanted(t *testing.T) {
	ds := []DomainState{{}, {}, {}}
	err := matchWanted(
		ds,
		[]DomainState{
			{Name: "a.example.com"},
			{Name: "b.example.com"},
			{Name: "a.example.com"},
		})
	if err == nil {
		t.Fatalf("Expected error")
	}
	if err.Error() != "repeated wanted domain: a.example.com" {
		t.Errorf("matchWanted() did detect find a repeated wanted domain. %s", err)
	}
}
