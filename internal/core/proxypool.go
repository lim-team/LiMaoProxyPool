package core

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/RussellLuo/timingwheel"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

type ProxyPool struct {
	opts        *Options
	ipChan      chan *IP // 爬取到的Ip
	checkIPChan chan *IP // 测速的IP
	stopChan    chan struct{}
	getters     []IGetter
	db          DB
	log.Log
	timingWheel *timingwheel.TimingWheel // Time wheel delay task
}

func NewProxyPool() *ProxyPool {
	opts := NewOptions()
	p := &ProxyPool{
		opts:        opts,
		ipChan:      make(chan *IP, 2000),
		checkIPChan: make(chan *IP, 2000),
		stopChan:    make(chan struct{}),
		db:          newMemoryDB(opts),
		Log:         log.NewTLog("ProxyPool"),
		timingWheel: timingwheel.NewTimingWheel(time.Millisecond*10, 100),
	}
	err := p.db.Load()
	if err != nil {
		panic(err)
	}
	p.getters = GetAllGetters(p)
	return p
}

func (p *ProxyPool) Run() {
	p.Info("运行中...")

	p.timingWheel.Start()

	// 循环测速
	go p.loopCheckSpeed()

	// 开多个协程去监听爬到的IP
	p.Info("开始新IP监听")
	for i := 0; i < p.opts.IPListenGoroutine; i++ {
		go p.listenIP()
	}

	// 开多个协程去监听需要测速的IP
	p.Info("开始测速监听")
	for i := 0; i < p.opts.SpeedCheckGoroutine; i++ {
		go p.listenSpeedCheckIP()
	}

	go p.runGetters()

	p.Schedule(time.Second*2, func() {
		count, _ := p.db.IPVaildCount()
		p.Info("有效IP数量", zap.Int("count", count))
	})

	p.Schedule(time.Second*20, func() {
		ips, _ := p.db.IPsVaild()
		for _, ip := range ips {
			fmt.Println("ip-->", ip)
		}
	})

	ticker := time.NewTicker(time.Minute * 10)
	saveTicker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			if p.needRunGetters() {
				// 爬取IP
				p.runGetters()
			}
		case <-saveTicker.C:
			err := p.db.Save()
			if err != nil {
				p.Error("保存数据失败！", zap.Error(err))
			}
		case <-p.stopChan:
			goto exit
		}
	}
exit:
	fmt.Println("end---->")
}

func (p *ProxyPool) Stop() {
	p.timingWheel.Stop()

	p.db.Close()

	close(p.stopChan)
}

func (p *ProxyPool) GetFastIP(ipTypes ...IPType) (*IP, error) {
	return p.db.IPFast(ipTypes...)
}

// Schedule 延迟任务
func (p *ProxyPool) Schedule(interval time.Duration, f func()) *timingwheel.Timer {
	return p.timingWheel.ScheduleFunc(&everyScheduler{
		Interval: interval,
	}, f)
}

func (p *ProxyPool) runGetters() {
	for _, getter := range p.getters {
		go getter.Run(p.ipChan)
	}
}

func (p *ProxyPool) needRunGetters() bool {

	return true
}

func (p *ProxyPool) listenIP() {

	for {
		select {
		case ip := <-p.ipChan:
			err := p.db.IPAddOrUpdate(ip)
			if err != nil {
				p.Error("添加或更新IP失败！", zap.Error(err), zap.String("ip", ip.Addr), zap.String("source", ip.Source))
				continue
			}
			p.checkIPChan <- ip
		case <-p.stopChan:
			p.Info("停止监听爬取IP")
			return
		}
	}
}

func (p *ProxyPool) listenSpeedCheckIP() {
	for {
		select {
		case ip := <-p.checkIPChan:
			if ip.SpeedChecking {
				continue
			}
			ip.SpeedChecking = true
			speed, err := p.speedCheck(ip)
			ip.SpeedChecking = false
			if err != nil {
				p.Warn("速度测试失败！", zap.Error(err), zap.String("ip", ip.Addr), zap.String("source", ip.Source))
				ip.Speed = p.opts.SpeedMax
				ip.LastSpeedAt = int(time.Now().Unix())
				ip.SpeedErrorCount++
			} else {
				ip.Speed = speed
				ip.SpeedErrorCount = 0
			}
			ip.LastSpeedAt = int(time.Now().Unix())
			if ip.SpeedErrorCount > 10 {
				p.db.IPRemove(ip)
			} else {
				err = p.db.IPAddOrUpdate(ip)
				if err != nil {
					p.Error("更新速度失败！", zap.Error(err), zap.String("ip", ip.Addr), zap.String("source", ip.Source))
					continue
				}
			}

		case <-p.stopChan:
			p.Info("停止监听测速IP")
			return
		}
	}
}

// 测速
func (p *ProxyPool) loopCheckSpeed() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			ips, err := p.db.IPsNeedCheck(p.opts.SpeedCheckInterval)
			if err != nil {
				p.Error("获取需要检查速度的IP集合失败！", zap.Error(err))
				continue
			}
			if len(ips) == 0 {
				continue
			}
			for _, ip := range ips {
				p.checkIPChan <- ip
			}
		case <-p.stopChan:
			p.Info("停止测速")
			return
		}
	}
}

func (p *ProxyPool) speedCheck(ip *IP) (int, error) {
	var pollURL string
	var testIP string
	if ip.Type == Https {
		testIP = "https://" + ip.Addr
		pollURL = "https://httpbin.org/get?show_env=1"
	} else if ip.Type == Socks4 {
		testIP = "socks4://" + ip.Addr
	} else if ip.Type == Socks5 {
		testIP = "socks5://" + ip.Addr
	} else {
		testIP = "http://" + ip.Addr
	}
	if pollURL == "" {
		pollURL = "http://httpbin.org/get?show_env=1"
	}
	proxy, _ := url.Parse(testIP)
	begin := time.Now()
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	netTransport := &http.Transport{
		Proxy:               http.ProxyURL(proxy),
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: 50,
	}
	httpClient := &http.Client{
		Timeout:   time.Second * 20,
		Transport: netTransport,
	}
	request, _ := http.NewRequest("GET", pollURL, nil)
	//设置一个header
	request.Header.Add("accept", "text/plain")
	resp, err := httpClient.Do(request)
	if err != nil {
		p.Warn("检查IP失败！", zap.Error(err), zap.String("testIP", testIP), zap.String("pollURL", pollURL))
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var resultMap map[string]interface{}
		err = json.Unmarshal(bodyBytes, &resultMap)
		if err == nil {
			speed := int(time.Now().Sub(begin).Nanoseconds() / 1000 / 1000)
			fmt.Println("speed->>>>>>>>>>>>>>>>>>", speed, ip.Addr, ip.Type)

			return speed, nil
		} else {
			p.Warn("测速返回内容错误！")
			fmt.Println("bodyBytes-->", string(bodyBytes))
			return 0, errors.New("测速返回内容错误！")
		}

	}
	p.Warn("检查IP失败！", zap.Error(errors.New("状态码错误！")), zap.Int("statusCode", resp.StatusCode), zap.String("testIP", testIP), zap.String("pollURL", pollURL))
	return 0, errors.New("状态码错误！")
}
