package main

import (
	"github.com/adshao/go-binance/v2/futures"
	"github.com/simon4545/binance-macd/bn"
)

const (
	apiKey    = "xzJKM9OUwYXxVrOpG9474d2Tgqx57QyABMzIekxXXzDSRNN5ClsNYlDblVVDqaNx"
	secretKey = "NG7W8uzFSu3PGnIx3lAyxIU232rhrQGsIz8n124A5eIlGeKHRnxKNji3V1cLgyzf"
)

var (
	client *futures.Client
)

func main() {
	// 初始化币安客户端
	client = futures.NewClient(apiKey, secretKey)
	bn.Init(client)
	select {}
}
