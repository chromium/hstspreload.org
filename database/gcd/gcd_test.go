package gcd

import "testing"
import "github.com/chromium/hstspreload.org/database/gcd"

func TestNewLocalBackend(t *testing.T) {
	_, shutdown, err := gcd.NewLocalBackend()
	if err != nil {
		t.Errorf("%s", err)
	}
	err = shutdown()
	if err != nil {
		t.Errorf("%s", err)
	}
}
