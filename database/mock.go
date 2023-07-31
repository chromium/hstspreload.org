package database

import "errors"

// Mock is a very simple Mock for our database.
type Mock struct {
	ds map[string]DomainState
	// This is a pointer so that we can pass around a Mock but continue
	// to control its behaviour.
	state *MockController
}

// MockController keeps track of mocking behaviour.
type MockController struct {
	FailCalls bool
}

// NewMock constructs a new mock, along with a MockController pointer to
// control the behaviour of the new Mock.
func NewMock() (m Mock, mc *MockController) {
	mc = &MockController{}
	m = Mock{
		ds:    map[string]DomainState{},
		state: mc,
	}
	return m, mc
}

// PutStates mock method
func (m Mock) PutStates(updates []DomainState, logf func(format string, args ...interface{})) error {
	if m.state.FailCalls {
		return errors.New("forced failure")
	}

	for _, s := range updates {
		m.PutState(s)
	}
	return nil
}

// PutState mock method
func (m Mock) PutState(update DomainState) error {
	if m.state.FailCalls {
		return errors.New("forced failure")
	}

	m.ds[update.Name] = update
	return nil
}

// StateForDomain mock method
func (m Mock) StateForDomain(domain string) (state DomainState, err error) {
	if m.state.FailCalls {
		return state, errors.New("forced failure")
	}

	s, ok := m.ds[domain]
	if ok {
		return s, nil
	}
	return DomainState{Status: StatusUnknown}, nil
}

// StatesForDomain mock method
func (m Mock) StatesForDomains(domains []string) (states []DomainState, err error) {
	if m.state.FailCalls {
		return states, errors.New("forced failure")
	}

	for _, domain := range domains {
		states = append(states, m.ds[domain])
	}
	return states, nil
}

// AllDomainStates mock method
func (m Mock) AllDomainStates() (states []DomainState, err error) {
	if m.state.FailCalls {
		return states, errors.New("forced failure")
	}

	for _, s := range m.ds {
		states = append(states, s)
	}
	return states, nil
}

// StatesWithStatus mock method
func (m Mock) StatesWithStatus(status PreloadStatus) (domains []DomainState, err error) {
	if m.state.FailCalls {
		return domains, errors.New("forced failure")
	}

	for _, s := range m.ds {
		if s.Status == status {
			domains = append(domains, s)
		}
	}
	return domains, nil
}
