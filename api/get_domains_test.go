package api

import (
	"reflect"
	"testing"

	"github.com/chromium/hstspreload/chromium/preloadlist"
)

var TestPreloadList = preloadlist.PreloadList{Entries: []preloadlist.Entry{
	{Name: "garron.net", Mode: "force-https", IncludeSubDomains: true, Policy: "bulk-18-weeks"},
	{Name: "example.com", Mode: "force-https", IncludeSubDomains: false, Policy: "bulk-18-weeks"},
	{Name: "gmail.com", Mode: "force-https", IncludeSubDomains: false, Policy: ""},
	{Name: "google.com", Mode: "", IncludeSubDomains: false, Policy: "custom"},
	{Name: "pinned.badssl.com", Mode: "", IncludeSubDomains: false, Policy: "bulk-1-year"}},
}

func TestGetDomainsByPolicyType(t *testing.T) {
	domains18weeks := GetDomainsByPolicyType(TestPreloadList, preloadlist.Bulk18Weeks)
	domains1year := GetDomainsByPolicyType(TestPreloadList, preloadlist.Bulk1Year)

	var expected18weeks = []preloadlist.Entry{
		{Name: "garron.net", Mode: "force-https", IncludeSubDomains: true, Policy: "bulk-18-weeks"},
		{Name: "example.com", Mode: "force-https", IncludeSubDomains: false, Policy: "bulk-18-weeks"},
	}

	var expected1year = []preloadlist.Entry{
		{Name: "pinned.badssl.com", Mode: "", IncludeSubDomains: false, Policy: "bulk-1-year"},
	}

	if !reflect.DeepEqual(domains18weeks, expected18weeks) {
		t.Errorf("bulk-18-week policy domains does not match expected")
	}

	if !reflect.DeepEqual(domains1year, expected1year) {
		t.Errorf("bulk-1-year policy domains does not match expected")
	}

}
