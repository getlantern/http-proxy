package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

const (
	maxDBKey  = "maxDB"
	redisAddr = "localhost:6379"
)

// Each client connects to a unique database. Database 0 holds a single key, maxDB, used to track
// the current highest database number.
func getNextDatabaseNumber() (int64, error) {
	c := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	nextDB, err := c.Incr(context.Background(), maxDBKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment and retrieve maxDB: %w", err)
	}
	return nextDB, nil
}

// TestRedis returns a client pointed at the local testing setup. This assumes the same setup
// specified in this project's test.bash file. Specifically, a Redis server should be listening on
// localhost:6379. The master name should be "mymaster" and TLS should be disabled.
//
// The database will be wiped before this function returns. The client will be closed when the test
// completes.
func TestRedis(t *testing.T) *redis.Client {
	t.Helper()

	nextDB, err := getNextDatabaseNumber()
	require.NoError(t, err)

	c := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   int(nextDB),
	})
	t.Cleanup(func() { c.Close() })
	return c
}
