package gcd

import (
	"fmt"
	"testing"
)

func ExampleNewLocalBackend() {
	_, shutdown, err := NewLocalBackend()
	if err != nil {
		fmt.Printf("%s", err)
	}
	defer shutdown()
}

func TestNewLocalBackend(t *testing.T) {
	_, shutdown, err := NewLocalBackend()
	if err != nil {
		t.Errorf("%s", err)
	}
	defer shutdown()
}
