package tlslistener

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
	"github.com/stretchr/testify/require"
)

func TestSessionTicketFormats(t *testing.T) {
	var key [32]byte
	_, err := rand.Read(key[:])
	require.NoError(t, err)
	standard := &tls.Config{}
	standard.SetSessionTicketKeys([][32]byte{key})
	legacy := &utls.Config{}
	legacy.SetSessionTicketKeys([][32]byte{key})
	connection := &clientHelloRecordingConn{cfg: standard, utlsCfg: legacy}
	for _, implementation := range []string{"go", "utls"} {
		t.Run(implementation, func(t *testing.T) {
			ticket := serverTicket(t, key, implementation)
			require.True(t, connection.validSessionTicket(ticket))
			ticket[len(ticket)-1] ^= 1
			require.False(t, connection.validSessionTicket(ticket))
			var otherKey [32]byte
			_, err := rand.Read(otherKey[:])
			require.NoError(t, err)
			require.False(t, connection.validSessionTicket(serverTicket(t, otherKey, implementation)))
		})
	}
}

func serverTicket(t *testing.T, key [32]byte, implementation string) []byte {
	t.Helper()
	serverRaw, clientRaw := net.Pipe()
	defer serverRaw.Close()
	defer clientRaw.Close()
	require.NoError(t, serverRaw.SetDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, clientRaw.SetDeadline(time.Now().Add(5*time.Second)))
	var server interface{ Handshake() error }
	if implementation == "go" {
		certificate, err := tls.LoadX509KeyPair("../test/data/server.crt", "../test/data/server.key")
		require.NoError(t, err)
		config := &tls.Config{Certificates: []tls.Certificate{certificate}, MaxVersion: tls.VersionTLS12}
		config.SetSessionTicketKeys([][32]byte{key})
		server = tls.Server(serverRaw, config)
	} else {
		certificate, err := utls.LoadX509KeyPair("../test/data/server.crt", "../test/data/server.key")
		require.NoError(t, err)
		config := &utls.Config{Certificates: []utls.Certificate{certificate}, MaxVersion: utls.VersionTLS12}
		config.SetSessionTicketKeys([][32]byte{key})
		server = utls.Server(serverRaw, config)
	}
	result := make(chan error, 1)
	go func() { result <- server.Handshake() }()
	cache := utls.NewLRUClientSessionCache(1)
	client := utls.Client(clientRaw, &utls.Config{
		ServerName: "example.com", InsecureSkipVerify: true,
		MaxVersion: utls.VersionTLS12, ClientSessionCache: cache,
	})
	require.NoError(t, client.Handshake())
	require.NoError(t, <-result)
	state, ok := cache.Get("example.com")
	require.True(t, ok)
	ticket, _, err := state.ResumptionState()
	require.NoError(t, err)
	require.NotEmpty(t, ticket)
	return ticket
}
