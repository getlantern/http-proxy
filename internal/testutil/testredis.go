package testutil

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// TestRedis returns a client pointed at the local testing setup. This assumes the same setup
// specified in this project's test.bash file. Specifically, a Redis server should be listening on
// localhost:6379. The master name should be "mymaster" and TLS should be in use.
//
// The database will be wiped before this function returns. The client will be closed when the test
// completes.
func TestRedis(t *testing.T) *redis.Client {
	t.Helper()

	c := redis.NewClient(&redis.Options{
		Addr:      "localhost:6379",
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	t.Cleanup(func() { c.Close() })
	wipeRedis(t, c)
	return c
}

func wipeRedis(t *testing.T, c *redis.Client) {
	host, _, err := net.SplitHostPort(c.Options().Addr)
	require.NoError(t, err)
	if host != "localhost" && host != "127.0.0.1" {
		t.Fatal("refusing to wipe non-local database")
	}

	allKeys, err := c.Keys(context.Background(), "*").Result()
	require.NoError(t, err)
	if len(allKeys) == 0 {
		return
	}
	_, err = c.Del(context.Background(), allKeys...).Result()
	require.NoError(t, err)
	currentKeys, err := c.Keys(context.Background(), "*").Result()
	require.NoError(t, err)
	require.Zero(t, len(currentKeys))
}
