package getter

import (
	"fmt"

	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	fmt.Println("init----->2")
	core.Register("kuaidaili", func(pp *core.ProxyPool) core.IGetter {
		return newKuaidaili(pp)
	})
}

type Kuaidaili struct {
	pp *core.ProxyPool
	log.Log
}

func newKuaidaili(pp *core.ProxyPool) *Kuaidaili {
	return &Kuaidaili{
		pp:  pp,
		Log: log.NewTLog("Kuaidaili"),
	}
}

func (k *Kuaidaili) Run(ipChan chan *core.IP) {
	pollURL := "http://www.kuaidaili.com/free/inha/"
	doc, _ := htmlquery.LoadURL(pollURL)
	if doc == nil {
		return
	}
	trNode, err := htmlquery.Find(doc, "//table[@class='table table-bordered table-striped']//tbody//tr")
	if err != nil {
		k.Warn("解析失败！", zap.Error(err))
	}
	for i := 0; i < len(trNode); i++ {
		tdNode, _ := htmlquery.Find(trNode[i], "//td")
		ip := htmlquery.InnerText(tdNode[0])
		port := htmlquery.InnerText(tdNode[1])
		Type := htmlquery.InnerText(tdNode[3])

		ipM := core.NewIP()
		ipM.Addr = ip + ":" + port
		if Type == "HTTPS" {
			ipM.Type = core.Https
		} else if Type == "HTTP" {
			ipM.Type = core.Http
		}
		ipM.Source = "KDL"
		ipChan <- ipM
	}

	k.Info("done")
}
