package database

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
	"google.golang.org/grpc"
)

const (
	// A blank project ID forces the project ID to be read from
	// the DATASTORE_PROJECT_ID environment variable.
	projectID = ""
)

/******** Backends ********/

type localDatastore struct {
	addr string
	cmd  *exec.Cmd
}

type prodDatastore struct{}

type datastoreBackend interface {
	newClient(ctx context.Context) (*datastore.Client, error)
}

/******** Port assignment for local backends ********/

var (
	portMutex sync.Mutex
	nextPort  = 9001
)

func portString() string {
	// TODO: Check that the port is available?
	portMutex.Lock()
	port := nextPort
	nextPort++
	portMutex.Unlock()

	return strconv.Itoa(port)
}

/******** localDatastore ********/

func newLocalDatastore() (db localDatastore, shutdown func(), err error) {
	db = localDatastore{}

	ps := portString()

	db.addr = "localhost:" + ps

	cmd := exec.Command(
		"java",
		"-cp",
		"./testing/gcd/CloudDatastore.jar",
		"com.google.cloud.datastore.emulator.CloudDatastore",
		"[datastore.go]",
		"start",
		"-p",
		ps, "--testing",
	)
	db.cmd = cmd

	err = cmd.Start()
	if err != nil {
		return db, func() {}, nil
	}

	shutdown = func() {
		cmd.Process.Kill()
	}

	// Wait for the server to start. 1000ms seems to work.
	time.Sleep(1000 * time.Millisecond)
	return db, shutdown, nil
}

// Based closely on datastore.NewClient().
func (db localDatastore) newClient(ctx context.Context) (*datastore.Client, error) {
	projectID := "hstspreload-local-testing"

	if db.addr == "" {
		return nil, errors.New("Empty addr. Uninitialized local backend?")
	}

	conn, err := grpc.Dial(db.addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("grpc.Dial: %v", err)
	}

	var o []cloud.ClientOption
	o = []cloud.ClientOption{cloud.WithBaseGRPC(conn)}
	client, err := datastore.NewClient(ctx, projectID, o...)
	if err != nil {
		return nil, err
	}
	return client, nil
}

/******** prodDatastore ********/

func newProdDatastore() (db prodDatastore) {
	// No special configuration in this case.
	return prodDatastore{}
}

func (db prodDatastore) newClient(ctx context.Context) (*datastore.Client, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return client, nil
}
