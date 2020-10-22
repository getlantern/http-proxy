package replicant

import (
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"sync"
	"testing"

	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	encodedServerConfig, err := ioutil.ReadFile("ReplicantServerConfig.txt")
	require.NoError(t, err)
	encodedClientConfig, err := ioutil.ReadFile("ReplicantClientConfig.txt")
	require.NoError(t, err)

	l, err = Wrap(l, string(encodedServerConfig))
	require.NoError(t, err)
	defer l.Close()
	log.Debugf("Listening at %v", l.Addr())

	go func() {
		for {
			conn, acceptErr := l.Accept()
			if acceptErr != nil {
				return
			}
			go io.Copy(conn, conn)
		}
	}()

	cfg, err := replicant.DecodeClientConfig(string(encodedClientConfig))
	require.NoError(t, err)
	dial := func(addr string) (net.Conn, error) {
		_conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		return replicant.NewClientConnection(_conn, *cfg)
	}

	iters := 1
	var wg sync.WaitGroup
	wg.Add(iters)
	for i := 0; i < iters; i++ {
		go func() {
			defer wg.Done()
			conn, err := dial(l.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			b := make([]byte, 2)
			rand.Read(b)
			_, err = conn.Write(b[:1])
			require.NoError(t, err)

			b2 := make([]byte, 1)
			_, err = io.ReadFull(conn, b2)
			require.NoError(t, err)
			assert.EqualValues(t, string(b[:1]), string(b2))

			w, err := conn.Write(b)
			require.NoError(t, err)
			b3 := make([]byte, len(b))
			r, err := io.ReadFull(conn, b3)
			require.NoError(t, err)
			assert.Equal(t, w, r)
			assert.EqualValues(t, string(b), string(b3))
		}()
	}

	wg.Wait()
}
