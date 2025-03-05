package main

import (
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

func main() {
	config := &configuration.Config{}
	config.Init()
	client = futures.NewClient(apiKey, secretKey)
	bn.Init(client, config)
	select {}
}
