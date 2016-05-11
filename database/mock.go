package database

import "errors"

// Mock is a very simple Mock for our database.
type Mock struct {
	ds map[string]DomainState
	// This is a pointer so that we can pass on Mock but continue
	// to control its behaviour.
	State *MockState
}

// MockState keeps track of mocking behaviour.
type MockState struct {
	FailCalls bool
}

// PutStates mock method
func (m Mock) PutStates(updates []DomainState, logf func(format string, args ...interface{})) error {
	if m.State.FailCalls == true {
		return errors.New("forced failure")
	}

	for _, s := range updates {
		m.PutState(s)
	}
	return nil
}

// PutState mock method
func (m Mock) PutState(update DomainState) error {
	if m.State.FailCalls == true {
		return errors.New("forced failure")
	}

	m.ds[update.Name] = update
	return nil
}

// StateForDomain mock method
func (m Mock) StateForDomain(domain string) (state DomainState, err error) {
	if m.State.FailCalls == true {
		return state, errors.New("forced failure")
	}

	s, ok := m.ds[domain]
	if ok {
		return s, nil
	}
	return DomainState{Status: StatusUnknown}, nil
}

// AllDomainStates mock method
func (m Mock) AllDomainStates() (states []DomainState, err error) {
	if m.State.FailCalls == true {
		return states, errors.New("forced failure")
	}

	for _, s := range m.ds {
		states = append(states, s)
	}
	return states, nil
}

// DomainsWithStatus mock method
func (m Mock) DomainsWithStatus(status PreloadStatus) (domains []string, err error) {
	if m.State.FailCalls == true {
		return domains, errors.New("forced failure")
	}

	for _, s := range m.ds {
		if s.Status == status {
			domains = append(domains, s.Name)
		}
	}
	return domains, nil
}
