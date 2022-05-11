package getter

import (
	"testing"

	"github.com/lim-team/LiMaoProxyPool/internal/core"
)

func TestIhuanRun(t *testing.T) {
	h := newIhuan()
	ipChan := make(chan *core.IP, 2000)
	h.Run(ipChan)
}
