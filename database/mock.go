package database

import "errors"

// Mock is a very simple Mock for our database.
type Mock struct {
	ds map[string]DomainState
	ids map[string]IneligibleDomainState
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
		ids:   map[string]IneligibleDomainState{},
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

func (m Mock) GetIneligibleDomainStates(domains []string) (states []IneligibleDomainState, err error) {
	if m.state.FailCalls {
		return states, errors.New("forced failure")
	}
	for _, domain := range domains {
		s, found := m.ids[domain]
		if found {
			states = append(states, s)
		} 
	}
	return states, nil
}

func (m Mock) SetIneligibleDomainStates(updates []IneligibleDomainState, logf func(format string, args ...interface{})) error {
	if m.state.FailCalls {
		return  errors.New("forced failure")
	}

	for _, update := range updates {
		m.ids[update.Name] = update
	}
	return nil
}

func (m Mock) DeleteIneligibleDomainStates(domains []string) (err error) {
    if m.state.FailCalls {
		return  errors.New("forced failure")
	}

	for _, domain := range domains {
		delete(m.ids, domain)
	}
	return nil
}

func (m Mock) GetAllIneligibleDomainStates() (states []IneligibleDomainState, err error) {
	if m.state.FailCalls {
		return states, errors.New("forced failure")
	}
	for _, s := range m.ids {
		states = append(states, s)
	}
	return states, nil
}
