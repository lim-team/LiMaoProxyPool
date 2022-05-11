package getter

import (
	"testing"

	"github.com/lim-team/LiMaoProxyPool/internal/core"
)

func TestHideMyRun(t *testing.T) {
	h := newHidemy()
	ipChan := make(chan *core.IP, 2000)
	h.Run(ipChan)
}
