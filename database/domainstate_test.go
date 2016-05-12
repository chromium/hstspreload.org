package database

import "testing"

func TestMatchWanted(t *testing.T) {
	ds := []DomainState{{}, {}, {}}
	err := MatchWanted(
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
