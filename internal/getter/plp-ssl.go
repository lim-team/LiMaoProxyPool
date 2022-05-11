package getter

import (
	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	core.Register("plp-ssl", func(pp *core.ProxyPool) core.IGetter {
		return newPLPSSL(pp)
	})
}

type PLPSSL struct {
	pp *core.ProxyPool
	log.Log
}

func newPLPSSL(pp *core.ProxyPool) *PLPSSL {
	return &PLPSSL{
		pp:  pp,
		Log: log.NewTLog("PLP-SSL"),
	}
}

func (p *PLPSSL) Run(ipChan chan *core.IP) {
	pollURL := "https://list.proxylistplus.com/SSL-List-1"
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

		IP.Source = "plpssl"
		ipChan <- IP
	}

	p.Info("done")
}
