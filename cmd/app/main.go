package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/judwhite/go-svc"
	"github.com/lim-team/LiMaoProxyPool/internal/core"
	_ "github.com/lim-team/LiMaoProxyPool/internal/getter"
)

type program struct {
	proxyPool *core.ProxyPool
}

func main() {
	prg := &program{}
	if err := svc.Run(prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	if env.IsWindowsService() {
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

func (p *program) Start() error {
	p.proxyPool = core.NewProxyPool()
	go p.proxyPool.Run()

	gp := goproxy.NewProxyHttpServer()
	gp.Verbose = true
	gp.ConnectDial = func(network, addr string) (net.Conn, error) {
		ip, err := p.proxyPool.GetFastIP(core.Https)
		if err != nil {
			fmt.Println("ConnectDial-获取proxy IP失败！->", err)
			return nil, nil
		}
		if ip == nil {
			fmt.Println("ConnectDial------------> no ip")
			return net.Dial(network, addr)
		}
		fmt.Println("ConnectDial------------>", ip)
		var proxyIP = p.proxyAddr(ip)
		conn, err := gp.NewConnectDialToProxy(proxyIP)(network, addr)
		if err != nil {
			fmt.Println("NewConnectDialToProxy------->", err)
			return nil, err
		}
		return conn, nil
	}
	gp.OnRequest().Do(goproxy.FuncReqHandler(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		fmt.Println("-------------request----------", req.URL.String())
		ip, err := p.proxyPool.GetFastIP()
		if err != nil {
			fmt.Println("获取proxy IP失败！->", err)
			return req, nil
		}
		fmt.Println("获取到proxy ip------------------->", ip)
		httpCli := p.newHttpClient(ip)
		newRequest, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
		if err != nil {
			fmt.Println("创建新的请求失败！-->", err)
			return req, nil
		}
		resp, err := httpCli.Do(newRequest)
		if err != nil {
			fmt.Println("请求失败！", err)
			return req, nil
		}
		return req, resp
	}))

	// py := proxy.NewServer()
	// proxy.Verbose = true
	go http.ListenAndServe(":8080", gp)
	return nil
}

func (p *program) proxyAddr(ip *core.IP) string {
	var proxyIP string
	if ip != nil {
		if ip.Type == "https" {
			proxyIP = "https://" + ip.Addr
		} else if ip.Type == "socks4" {
			proxyIP = "socks4://" + ip.Addr
		} else if ip.Type == "socks5" {
			proxyIP = "socks5://" + ip.Addr
		} else {
			proxyIP = "http://" + ip.Addr
		}
	}
	return proxyIP

}

func (p *program) newHttpClient(ip *core.IP) *http.Client {
	var proxyIP = p.proxyAddr(ip)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	netTransport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: 50,
	}
	if proxyIP != "" {
		proxy, _ := url.Parse(proxyIP)
		netTransport.Proxy = http.ProxyURL(proxy)
	}
	return &http.Client{
		Timeout:   time.Second * 20,
		Transport: netTransport,
	}
}

func (p *program) Stop() error {
	p.proxyPool.Stop()
	return nil
}
