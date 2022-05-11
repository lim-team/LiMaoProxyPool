package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lim-team/LiMaoProxyPool/internal/core"
	"github.com/lim-team/LiMaoProxyPool/pkg/log"
	"go.uber.org/zap"
)

type halfClosable interface {
	net.Conn
	CloseWrite() error
	CloseRead() error
}

var _ halfClosable = (*net.TCPConn)(nil)

type Server struct {
	log.Log
	proxyPool *core.ProxyPool
}

func NewServer(proxyPool *core.ProxyPool) *Server {
	return &Server{
		Log:       log.NewTLog("Server"),
		proxyPool: proxyPool,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("r.Method--->", r.Method)
	fmt.Println("request-------->", r.URL.String())
	if r.Method == "CONNECT" {
		s.handleHttps(w, r)
		return
	}
	if !r.URL.IsAbs() || strings.HasPrefix(r.Host, "127.0.0.1") {
		w.Write([]byte("this is proxy"))
		return
	}

	// trans := http.Transport{
	// 	Proxy: func(req *http.Request) (*url.URL, error) {
	// 		return url.Parse("http://localhost:8082")
	// 	},
	// 	DialContext: (&net.Dialer{
	// 		Timeout:   30 * time.Second,
	// 		KeepAlive: 30 * time.Second,
	// 	}).DialContext,
	// 	ForceAttemptHTTP2:     true,
	// 	MaxIdleConns:          100,
	// 	IdleConnTimeout:       90 * time.Second,
	// 	TLSHandshakeTimeout:   10 * time.Second,
	// 	ExpectContinueTimeout: 1 * time.Second,
	// }
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		s.Error("error-------->", zap.Error(err))
	}

	fmt.Println("resp-------->", resp)
	fmt.Println("w.Header()-------->", w)
	copyHeaders(w.Header(), resp.Header, true)
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		s.Error("复制body失败！", zap.Error(err))
	}
	if err := resp.Body.Close(); err != nil {
		s.Error("关闭body失败！", zap.Error(err))
	}
}

func (s *Server) newHttpClient(ip *core.IP) *http.Client {
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

func (s *Server) handleHttps(w http.ResponseWriter, r *http.Request) {
	hij, ok := w.(http.Hijacker)
	if !ok {
		panic("httpserver does not support hijacking")
	}
	proxyClient, _, e := hij.Hijack()
	if e != nil {
		panic("Cannot hijack connection " + e.Error())
	}
	host := r.URL.Host

	targetSiteCon, err := net.Dial("tcp", host)
	if err != nil {
		s.Error("建立连接错误！", zap.Error(err))
		return
	}
	s.Info("ccepting CONNECT to ", zap.String("host", host))
	proxyClient.Write([]byte("HTTP/1.0 200 Connection established\r\n\r\n"))
	targetTCP, targetOK := targetSiteCon.(halfClosable)
	proxyClientTCP, clientOK := proxyClient.(halfClosable)
	if targetOK && clientOK {
		go s.copyAndClose(targetTCP, proxyClientTCP)
		go s.copyAndClose(proxyClientTCP, targetTCP)
	} else {
		go func() {
			var wg sync.WaitGroup
			wg.Add(2)
			go s.copyOrWarn(targetSiteCon, proxyClient, &wg)
			go s.copyOrWarn(proxyClient, targetSiteCon, &wg)
			wg.Wait()
			proxyClient.Close()
			targetSiteCon.Close()

		}()
	}
}

func (s *Server) copyOrWarn(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	if _, err := io.Copy(dst, src); err != nil {
		s.Warn("Error copying to client", zap.Error(err))
	}
	wg.Done()
}
func (s *Server) copyAndClose(dst, src halfClosable) {
	if _, err := io.Copy(dst, src); err != nil {
		s.Warn("Error copying to client", zap.Error(err))
	}

	dst.CloseWrite()
	src.CloseRead()
}

func copyHeaders(dst, src http.Header, keepDestHeaders bool) {
	fmt.Println("dest-->", dst, src)
	if !keepDestHeaders {
		for k := range dst {
			dst.Del(k)
		}
	}
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}
