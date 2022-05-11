package getter

import (
	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	core.Register("plp", func(pp *core.ProxyPool) core.IGetter {
		return newPLP(pp)
	})
}

type PLP struct {
	pp *core.ProxyPool
	log.Log
}

func newPLP(pp *core.ProxyPool) *PLP {
	return &PLP{
		pp:  pp,
		Log: log.NewTLog("PLP"),
	}
}

func (p *PLP) Run(ipChan chan *core.IP) {
	pollURL := "https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-1"
	doc, _ := htmlquery.LoadURL(pollURL)
	if doc == nil {
		return
	}
	trNode, err := htmlquery.Find(doc, "//div[@class='hfeed site']//table[@class='bg']//tbody//tr")
	if err != nil {
		p.Warn("解析失败", zap.Error(err))
	}
	for i := 3; i < len(trNode); i++ {
		tdNode, _ := htmlquery.Find(trNode[i], "//td")
		ip := htmlquery.InnerText(tdNode[1])
		port := htmlquery.InnerText(tdNode[2])
		Type := htmlquery.InnerText(tdNode[6])

		IP := core.NewIP()
		IP.Addr = ip + ":" + port

		if Type == "yes" {
			IP.Type = core.Https

		} else if Type == "no" {
			IP.Type = core.Http
		}

		IP.Source = "plp"
		ipChan <- IP
	}

	p.Info("done")
}
