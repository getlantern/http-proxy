// Package testredis provides an embedded Redis instance for unit testing
package testredis

import (
	"fmt"
	"io/ioutil"
	"os"

	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/server"

	"gopkg.in/redis.v3"
)

// Redis is a testing Redis
type Redis interface {
	// Addr gets the address of this Redis
	Addr() string

	// Client opens a new client to this Redis
	Client() *redis.Client

	// Close closes this Redis and associated resources
	Close() error
}

type testRedis struct {
	dir string
	app *server.App
}

func (tr *testRedis) Addr() string {
	return tr.app.Address()
}

func (tr *testRedis) Client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: tr.app.Address(),
	})
}

func (tr *testRedis) Close() error {
	tr.app.Close()
	return os.RemoveAll(tr.dir)
}

func Open() (Redis, error) {
	tmpDir, err := ioutil.TempDir("", "redis")
	if err != nil {
		return nil, fmt.Errorf("Unable to create temp dir: %v", err)
	}

	cfg := lediscfg.NewConfigDefault()
	cfg.DataDir = tmpDir

	app, err := server.NewApp(cfg)
	if err != nil {
		return nil, err
	}
	go app.Run()

	return &testRedis{tmpDir, app}, nil
}
