// Package testredis provides an embedded Redis instance for unit testing
package testredis

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/server"

	"gopkg.in/redis.v3"
)

type testRedis struct {
	dir string
	app *server.App
}

func (tr *testRedis) Close() error {
	tr.app.Close()
	return os.RemoveAll(tr.dir)
}

func Open() (io.Closer, *redis.Client, error) {
	tmpDir, err := ioutil.TempDir("", "redis")
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to create temp dir: %v", err)
	}

	cfg := lediscfg.NewConfigDefault()
	cfg.DataDir = tmpDir

	app, err := server.NewApp(cfg)
	if err != nil {
		return nil, nil, err
	}
	go app.Run()
	rc := redis.NewClient(&redis.Options{
		Addr: app.Address(),
	})

	return &testRedis{tmpDir, app}, rc, nil
}
