// +build linux

package bbr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManageUpstreamABE(t *testing.T) {
	bm := &middleware{}
	bm.setUpstreamABE(5000.001)
	assert.EqualValues(t, 5000.001, bm.getUpstreamABE())
}
