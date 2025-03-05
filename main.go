package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/simon4545/binance-macd/bn"
	"github.com/simon4545/binance-macd/configuration"
)

const (
	apiKey    = "xzJKM9OUwYXxVrOpG9474d2Tgqx57QyABMzIekxXXzDSRNN5ClsNYlDblVVDqaNx"
	secretKey = "NG7W8uzFSu3PGnIx3lAyxIU232rhrQGsIz8n124A5eIlGeKHRnxKNji3V1cLgyzf"
)

var (
	client *futures.Client
)

var weightMaxPerMinute int
var usedWeight int

// transport binance transport client
type transport struct {
	UnderlyingTransport http.RoundTripper
}

// RoundTrip implement http roundtrip
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.UnderlyingTransport.RoundTrip(req)
	if resp != nil && resp.Header != nil {
		usedWeight, _ = strconv.Atoi(resp.Header.Get("X-Mbx-Used-Weight-1m"))
		//如果请求次数超过900，开始打印日志
		if usedWeight > 900 {
			log.Println("request count:::", usedWeight)
		}
	}

	return resp, err
}
func CheckRateLimit() {
	if usedWeight > weightMaxPerMinute {
		now := time.Now()
		time.Sleep(time.Until(now.Add(time.Duration(60-now.Second()) * time.Second)))
	}
}
func main() {
	config := &configuration.Config{}
	config.Init()
	client = futures.NewClient(apiKey, secretKey)
	futures.WebsocketKeepalive = true
	// binance.WebsocketTimeout = time.Second * 1
	//校准时间
	client.NewSetServerTimeService().Do(context.Background())
	//设置这个不会报API超过限制
	c := http.Client{Transport: &transport{UnderlyingTransport: http.DefaultTransport}}
	client.HTTPClient = &c
	bn.Init(client, config)
	select {}
}
