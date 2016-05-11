package database

// Mock is a very simple Mock for our database.
type Mock map[string]DomainState

// PutStates mock method
func (m Mock) PutStates(updates []DomainState, logf func(format string, args ...interface{})) error {
	for _, s := range updates {
		m.PutState(s)
	}
	return nil
}

// PutState mock method
func (m Mock) PutState(update DomainState) error {
	m[update.Name] = update
	return nil
}

// StateForDomain mock method
func (m Mock) StateForDomain(domain string) (state DomainState, err error) {
	s, ok := m[domain]
	if ok {
		return s, nil
	}
	return DomainState{Status: StatusUnknown}, nil
}

// AllDomainStates mock method
func (m Mock) AllDomainStates() (states []DomainState, err error) {
	for _, s := range m {
		states = append(states, s)
	}
	return states, nil
}

// DomainsWithStatus mock method
func (m Mock) DomainsWithStatus(status PreloadStatus) (domains []string, err error) {
	for _, s := range m {
		if s.Status == status {
			domains = append(domains, s.Name)
		}
	}
	return domains, nil
}
