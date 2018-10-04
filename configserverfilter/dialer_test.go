package configserverfilter

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	"github.com/getlantern/mockconn"
	"github.com/stretchr/testify/assert"
)

func TestDialerConfigServer(t *testing.T) {
	domains := make([]string, 0)
	domains = append(domains, "config.getiantem.org")
	opts := &Options{
		AuthToken: "",
		Domains:   domains,
	}
	d := &net.Dialer{}
	dial := Dialer(d.DialContext, opts)
	conn, err := dial(context.Background(), "tcp", "config.getiantem.org:443")
	assert.NoError(t, err)
	conn.Close()
}

func TestDialer(t *testing.T) {
	dummyDial := func(ctx context.Context, net, addr string) (net.Conn, error) {
		return mockconn.SucceedingDialer([]byte{}).Dial(net, addr)
	}
	d := Dialer(dummyDial, &Options{"", []string{"site1", "site2"}})

	c, _ := d(context.Background(), "tcp", "site1")
	_, ok := c.(*tls.Conn)
	assert.True(t, ok, "should override dialer if site is in list")
	c, _ = d(context.Background(), "tcp", "other")
	_, ok = c.(*tls.Conn)
	assert.False(t, ok, "should not override dialer for other dialers")
}
