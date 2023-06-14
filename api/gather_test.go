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

var expected18weeks = []Domain{"garron.net", "example.com"}
var expected1year = []Domain{"pinned.badssl.com"}

func TestGetBulk18WeeksDomains(t *testing.T) {
	domains18weeks := GetBulk18WeeksDomains(TestPreloadList)

	if !reflect.DeepEqual(domains18weeks, expected18weeks) {
		t.Errorf("bulk18week domains does not match expected")
	}
}

func TestGetBulk1YearDomains(t *testing.T) {
	domains1year := GetBulk1YearDomains(TestPreloadList)

	if !reflect.DeepEqual(domains1year, expected1year) {
		t.Errorf("bulk1year domains does not match")
	}
}
