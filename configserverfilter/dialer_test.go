package configserverfilter

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/getlantern/mockconn"
	"github.com/stretchr/testify/assert"
)

func TestDialerConfigServer(t *testing.T) {
	domains := make([]string, 0)
	domains = append(domains, "config.getiantem.org")
	opts := &Options{
		AuthToken:          "",
		Domains:            domains,
		ClientIPCacheClear: time.Hour * 2,
	}
	dial := Dialer(net.Dial, opts)
	conn, err := dial("tcp", "config.getiantem.org:443")
	assert.NoError(t, err)
	conn.Close()
}

func TestDialer(t *testing.T) {
	var address string
	dummyDial := func(net, addr string) (net.Conn, error) {
		address = addr
		return mockconn.SucceedingDialer([]byte{}).Dial(net, addr)
	}
	d := Dialer(dummyDial, &Options{"", []string{"site1", "site2"}, 0})

	c, _ := d("tcp", "site1")
	_, ok := c.(*tls.Conn)
	assert.True(t, ok, "should override dialer if site is in list")
	c, _ = d("tcp", "other")
	_, ok = c.(*tls.Conn)
	assert.False(t, ok, "should not override dialer for other dialers")
}
