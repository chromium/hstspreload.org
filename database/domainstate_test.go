package database

import "testing"

func TestMatchWanted(t *testing.T) {
	ds := []DomainState{{}, {}, {}}
	if MatchWanted(
		ds,
		[]DomainState{
			{Name: "a.example.com"},
			{Name: "b.example.com"},
			{Name: "a.example.com"},
		}) {
		t.Fatalf("Expected false")
	}
}
