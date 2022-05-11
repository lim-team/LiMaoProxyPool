package getter

import (
	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	core.Register("ip3306", func(pp *core.ProxyPool) core.IGetter {
		return newIP3306()
	})
}

type IP3306 struct {
	log.Log
}

func newIP3306() *IP3306 {

	return &IP3306{
		Log: log.NewTLog("IP3306"),
	}
}

func (g *IP3306) Run(ipChan chan *core.IP) {
	pollURL := "http://www.ip3366.net/free/?stype=1&page=1"
	doc, _ := htmlquery.LoadURL(pollURL)
	trNode, err := htmlquery.Find(doc, "//div[@id='list']//table//tbody//tr")
	if err != nil {
		g.Warn("解析失败", zap.Error(err))
	}
	//debug begin
	for i := 1; i < len(trNode); i++ {
		tdNode, _ := htmlquery.Find(trNode[i], "//td")
		ip := htmlquery.InnerText(tdNode[0])
		port := htmlquery.InnerText(tdNode[1])
		Type := htmlquery.InnerText(tdNode[3])

		IP := core.NewIP()
		IP.Addr = ip + ":" + port

		if Type == "HTTPS" {
			IP.Type = core.Https

		} else if Type == "HTTP" {
			IP.Type = core.Http
		}
		IP.Source = "ip3366.net"
		ipChan <- IP

	}
	g.Info("done.")
}
