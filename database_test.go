package main

import "testing"

func TestAllDomainStates(t *testing.T) {

	db, shutdown, err := newLocalDatastore()
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	defer shutdown()

	putStates(
		db,
		[]DomainState{{Name: "garron.net", Status: "preloaded"}},
		func(format string, args ...interface{}) {},
	)

	// _, err = datastoreStatesForQuery(ctx, client, datastore.NewQuery("DomainState"))
	// if err != nil {
	// 	t.Errorf("cannot get all domain states %s", err)
	// 	return
	// }
	// t.Errorf("%#v", states)

	// if len(states) != 0 {
	// 	t.Errorf("database should start empty", states)
	// }
}
