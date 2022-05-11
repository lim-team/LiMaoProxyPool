package getter

import (
	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	core.Register("ihuan", func(pp *core.ProxyPool) core.IGetter {
		return newIhuan()
	})
}

type ihuan struct {
	log.Log
}

func newIhuan() *ihuan {
	return &ihuan{
		Log: log.NewTLog("ihuan"),
	}
}

func (g *ihuan) Run(ipChan chan *core.IP) {
	pollURL := "https://ip.ihuan.me/"
	doc, _ := htmlquery.LoadURL(pollURL)
	if doc == nil {
		return
	}
	trNode, err := htmlquery.Find(doc, "//div[@class='table-responsive']//table//tbody//tr")
	if err != nil {
		g.Warn("解析失败", zap.Error(err))
	}
	for i := 1; i < len(trNode); i++ {
		tdNode, _ := htmlquery.Find(trNode[i], "//td")
		ip := htmlquery.InnerText(tdNode[0])
		port := htmlquery.InnerText(tdNode[1])
		typ := htmlquery.InnerText(tdNode[4])

		IP := core.NewIP()
		IP.Addr = ip + ":" + port
		if typ == "支持" {
			IP.Type = core.Https
		} else {
			IP.Type = core.Http
		}
		IP.Source = "ip.ihuan.me"
		ipChan <- IP
	}
}
