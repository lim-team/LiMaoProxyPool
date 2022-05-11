package getter

import (
	"strings"

	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	core.Register("hidemy", func(pp *core.ProxyPool) core.IGetter {
		return newHidemy()
	})
}

type hidemy struct {
	log.Log
}

func newHidemy() *hidemy {
	return &hidemy{
		Log: log.NewTLog("hidemy"),
	}
}

func (g *hidemy) Run(ipChan chan *core.IP) {
	pollURL := "https://hidemy.name/cn/proxy-list/?type=s#list"
	doc, _ := htmlquery.LoadURL(pollURL)
	if doc == nil {
		return
	}
	trNode, err := htmlquery.Find(doc, "//div[@class='table_block']//table//tbody//tr")
	if err != nil {
		g.Warn("解析失败", zap.Error(err))
	}
	for i := 1; i < len(trNode); i++ {
		tdNode, _ := htmlquery.Find(trNode[i], "//td")
		ip := htmlquery.InnerText(tdNode[0])
		port := htmlquery.InnerText(tdNode[1])
		Type := htmlquery.InnerText(tdNode[4])

		IP := core.NewIP()
		IP.Addr = ip + ":" + port

		types := strings.Split(Type, ",")

		for _, typ := range types {
			if typ == "HTTPS" {
				IP.Type = core.Https

			} else if typ == "HTTP" {
				IP.Type = core.Http
			}
			IP.Source = "hidemy.name"
			ipChan <- IP
		}

	}
	g.Info("done.")

}
