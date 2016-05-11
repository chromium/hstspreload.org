package gcd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
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

	numLocalProbes    = 10
	initialProbeSleep = 300 * time.Millisecond
	localProbeSpacing = 100 * time.Millisecond
)

/******** Backends ********/

// LocalBackend represents an emulated Google Cloud Datastore
// running on localhost
type LocalBackend struct {
	// unexported fields
	addr string
	cmd  *exec.Cmd
}

// ProdBackend represents the production instance of
// Google Cloud Datastore
type ProdBackend struct{}

// Backend is an abstraction over {Local, Prod}Datastore
// that allows callers to construct a new client without having to
// know about whether it's local.
type Backend interface {
	NewClient(ctx context.Context) (*datastore.Client, error)
}

/******** Port assignment for local backends ********/

func portString() (string, error) {
	// Ask for a port to listen on.
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}

	// Give up the port and return it as a (very likely) free port.
	l.Close()
	parts := strings.Split(l.Addr().String(), ":")
	ps := parts[len(parts)-1]

	return ps, nil
}

/******** LocalBackend ********/

// NewLocalBackend spawns a new LocalBackend using Java.
// When there is no error, make sure to call shutdown() in order to
// terminate the Java process.
func NewLocalBackend() (db LocalBackend, shutdown func() error, err error) {
	db = LocalBackend{}
	shutdown = func() error { return nil }

	ps, err := portString()
	if err != nil {
		return db, shutdown, err
	}
	db.addr = "localhost:" + ps

	jarPath := path.Join(os.Getenv("HOME"), ".datastore-emulator", "gcd", "CloudDatastore.jar")
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return db, shutdown, fmt.Errorf("Datastore emulator does not exist: %s", err)
	}

	cmd := exec.Command(
		"java",
		"-cp",
		jarPath,
		"com.google.cloud.datastore.emulator.CloudDatastore",
		"[gcd.go]",
		"start",
		"-p",
		ps,
		"--testing",
	)
	db.cmd = cmd

	err = cmd.Start()
	if err != nil {
		return db, shutdown, err
	}

	shutdown = func() error {
		return cmd.Process.Kill()
	}

	time.Sleep(initialProbeSleep)
	for i := 0; i < numLocalProbes; i++ {
		time.Sleep(localProbeSpacing)
		resp, err := http.Get("http://" + db.addr)
		if err == nil {
			if resp.StatusCode != 200 {
				return db, shutdown, fmt.Errorf("Wrong status code: %d", resp.StatusCode)
			}
			return db, shutdown, nil
		}
		if !strings.Contains(err.Error(), "connection refused") {
			return db, shutdown, err
		}
	}

	return db, shutdown, fmt.Errorf("could not connect")
}

// NewClient constructs a datastore client for the emulated LocalBackend.
// The constructed client will work offline and never connect to the wide internet.
func (db LocalBackend) NewClient(ctx context.Context) (*datastore.Client, error) {
	projectID := "hstspreload-local-testing"

	// The code below is based closely on the implementation of
	//  datastore.NewClient().

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

// Reset resets the local backend to an empty database.
func (db LocalBackend) Reset() error {
	resp, err := http.Post("http://"+db.addr+"/reset", "text/plain", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Could not clear local datastore. Unexpected status: %d", 200)
	}
	return nil
}

/******** ProdBackend ********/

// NewProdBackend constructs a new ProdBackend.
func NewProdBackend() (db ProdBackend) {
	// No special configuration in this case.
	return ProdBackend{}
}

// NewClient is a wrapper around the default implementation of
// datastore.NewClient(), calling out to the real, live
// Google Cloud Datastore.
func (db ProdBackend) NewClient(ctx context.Context) (*datastore.Client, error) {
	return datastore.NewClient(ctx, projectID)
}
