package api

import (
	"reflect"
	"testing"

	"github.com/chromium/hstspreload/chromium/preloadlist"
)

func TestGatherLists(t *testing.T) {
	TestPreloadList := preloadlist.PreloadList{Entries: []preloadlist.Entry{
		{Name: "garron.net", Mode: "force-https", IncludeSubDomains: true, Policy: "bulk-18-weeks"},
		{Name: "example.com", Mode: "force-https", IncludeSubDomains: false, Policy: "bulk-18-weeks"},
		{Name: "gmail.com", Mode: "force-https", IncludeSubDomains: false, Policy: ""},
		{Name: "google.com", Mode: "", IncludeSubDomains: false, Policy: "custom"},
		{Name: "pinned.badssl.com", Mode: "", IncludeSubDomains: false, Policy: "bulk-1-year"}},
	}

	domains18weeks, domains1year := gatherLists(TestPreloadList)

	expected18weeks := []string{"garron.net", "example.com"}
	expected1year := []string{"pinned.badssl.com"}
	if !reflect.DeepEqual(domains18weeks, expected18weeks) {
		t.Errorf("bulk18week domains does not match expected")
	}
	if !reflect.DeepEqual(domains1year, expected1year) {
		t.Errorf("bulk1year domains does not match")
	}

}
