package api

import (
	"errors"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload/chromium/preloadlist"
)

type hstspreloadWrapper interface {
	PreloadableDomain(string) (*string, hstspreload.Issues)
	RemovableDomain(string) (*string, hstspreload.Issues)
}

type preloadlistWrapper interface {
	NewFromLatest() (preloadlist.PreloadList, error)
}

/******** actual ********/

type actualHstspreload struct{}
type actualPreloadlist struct{}

func (actualHstspreload) PreloadableDomain(domain string) (*string, hstspreload.Issues) {
	return hstspreload.PreloadableDomain(domain)
}
func (actualHstspreload) RemovableDomain(domain string) (*string, hstspreload.Issues) {
	return hstspreload.RemovableDomain(domain)
}
func (actualPreloadlist) NewFromLatest() (preloadlist.PreloadList, error) {
	return preloadlist.NewFromLatest()
}

/******** mock ********/

type mockHstspreload struct {
	// The mock will return verdicts based on these maps.
	// Remember that you must `make` a map before adding values:  https://blog.golang.org/go-maps-in-action#TOC_2.
	preloadableResponses map[string]hstspreload.Issues
	removableResponses   map[string]hstspreload.Issues
}
type mockPreloadlist struct {
	list      preloadlist.PreloadList
	failCalls bool
}

func (h mockHstspreload) PreloadableDomain(domain string) (*string, hstspreload.Issues) {
	return nil, h.preloadableResponses[domain]
}
func (h mockHstspreload) RemovableDomain(domain string) (*string, hstspreload.Issues) {
	return nil, h.removableResponses[domain]
}
func (c mockPreloadlist) NewFromLatest() (preloadlist.PreloadList, error) {
	if c.failCalls {
		return preloadlist.PreloadList{}, errors.New("forced failure")
	}
	return c.list, nil
}
