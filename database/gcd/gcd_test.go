package gcd

import "testing"

func TestNewLocalBackend(t *testing.T) {
	_, shutdown, err := NewLocalBackend()
	if err != nil {
		t.Errorf("%s", err)
	}
	err = shutdown()
	if err != nil {
		t.Errorf("%s", err)
	}
}
