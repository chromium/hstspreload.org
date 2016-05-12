package api

import (
	"errors"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload/chromiumpreload"
)

type hstspreloadWrapper interface {
	PreloadableDomain(string) (*string, hstspreload.Issues)
	RemovableDomain(string) (*string, hstspreload.Issues)
}

type chromiumpreloadWrapper interface {
	GetLatest() (chromiumpreload.PreloadList, error)
}

/******** actual ********/

type actualHstspreload struct{}
type actualChromiumpreload struct{}

func (actualHstspreload) PreloadableDomain(domain string) (*string, hstspreload.Issues) {
	return hstspreload.PreloadableDomain(domain)
}
func (actualHstspreload) RemovableDomain(domain string) (*string, hstspreload.Issues) {
	return hstspreload.RemovableDomain(domain)
}
func (actualChromiumpreload) GetLatest() (chromiumpreload.PreloadList, error) {
	return chromiumpreload.GetLatest()
}

/******** mock ********/

type mockHstspreload struct {
	// The mock will return verdicts based on these maps.
	// Remember that you must `make` a map before adding values:  https://blog.golang.org/go-maps-in-action#TOC_2.
	preloadableResponses map[string]hstspreload.Issues
	removableResponses   map[string]hstspreload.Issues
}
type mockChromiumpreload struct {
	list      chromiumpreload.PreloadList
	failCalls bool
}

func (h mockHstspreload) PreloadableDomain(domain string) (*string, hstspreload.Issues) {
	return nil, h.preloadableResponses[domain]
}
func (h mockHstspreload) RemovableDomain(domain string) (*string, hstspreload.Issues) {
	return nil, h.removableResponses[domain]
}
func (c mockChromiumpreload) GetLatest() (chromiumpreload.PreloadList, error) {
	if c.failCalls {
		return chromiumpreload.PreloadList{}, errors.New("forced failure")
	}
	return c.list, nil
}
