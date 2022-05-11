package getter

import (
	"fmt"
	"strings"

	"github.com/Aiicy/htmlquery"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

func init() {
	fmt.Println("init----->")
	core.Register("pzzqz", func(pp *core.ProxyPool) core.IGetter {
		return newPZZQZ(pp)
	})
}

type PZZQZ struct {
	pp *core.ProxyPool
	log.Log
}

func newPZZQZ(pp *core.ProxyPool) *PZZQZ {
	return &PZZQZ{
		pp:  pp,
		Log: log.NewTLog("PZZQZ"),
	}
}
func (p *PZZQZ) Run(ipChan chan *core.IP) {
	p.Info("开始获取IP")
	pollURL := "http://pzzqz.com/"
	doc, _ := htmlquery.LoadURL(pollURL)
	if doc == nil {
		return
	}
	trNode, err := htmlquery.Find(doc, "//table[@class='table table-hover']//tbody//tr")
	if err != nil {
		p.Warn("解析失败！", zap.Error(err))
	}
	for i := 0; i < len(trNode); i++ {
		tdNode, _ := htmlquery.Find(trNode[i], "//td")
		ip := htmlquery.InnerText(tdNode[0])
		port := htmlquery.InnerText(tdNode[1])
		types := htmlquery.InnerText(tdNode[4])
		if len(types) == 0 {
			continue
		}
		typeArray := strings.Split(types, ",")

		for _, typeS := range typeArray {
			ipM := core.NewIP()
			ipM.Addr = ip + ":" + port
			ipM.Type = core.IPType(strings.ToLower(typeS))
			ipM.Source = "pzzqz"
			ipChan <- ipM
		}

	}
	p.Info("done")

}
