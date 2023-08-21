package tlslistener

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRotateSessionTicketKeysInMemory(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)

	_, err := rand.Read(key1)
	require.NoError(t, err)
	_, err = rand.Read(key2)
	require.NoError(t, err)

	initialKeyBytes := append(key1, key2...)
	rotatedKeyBytes := rotateSessionTicketKeysInMemory(initialKeyBytes)
	require.EqualValues(t, append(key2, key1...), rotatedKeyBytes)

	rotatedKeyBytes = rotateSessionTicketKeysInMemory(rotatedKeyBytes)
	require.EqualValues(t, initialKeyBytes, rotatedKeyBytes)
}
