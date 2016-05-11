package database

import (
	"testing"

	"github.com/chromium/hstspreload.appspot.com/database/gcd"
)

func TestAllDomainStates(t *testing.T) {

	db, shutdown, err := gcd.NewLocalBackend()
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	defer shutdown()

	PutStates(
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
