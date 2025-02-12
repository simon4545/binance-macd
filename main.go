package main

import (
	"os"
	"time"

	"github.com/simon4545/binance-macd/bn"
	"github.com/simon4545/binance-macd/configuration"
	"github.com/simon4545/binance-macd/db"
)

var symbols []string
var config *configuration.Config

func check() {
	currentTime := time.Now()
	targetYear := time.Date(2026, time.June, 1, 0, 0, 0, 0, currentTime.Location())
	if currentTime.After(targetYear) {
		os.Exit(0)
	}
}
func init() {
	check()

	configuration.LotSizeMap = make(map[string]float64)
	configuration.PriceFilterMap = make(map[string]float64)
	configuration.AtrMap = make(map[string]float64)
	configuration.FeeMap = make(map[string]float64)

}

func main() {
	// InitWS()
	config = &configuration.Config{}
	config.Init()
	db.InitDB()
	go bn.Init(config)
	go WebInit()
	// go strategy.Run(config)
	select {}
}

// func list() {
// 	url := "https://api.binance.com/api/v3/ticker/24hr"
// 	// url := "https://api.binance.com/api/v3/ticker/24hr"
// 	response, err := http.Get(url)
// 	if err != nil {
// 		log.Println("Error making GET request:", err)
// 		return
// 	}
// 	defer response.Body.Close()
// 	bodyBytes, err := io.ReadAll(response.Body)
// 	if err != nil {
// 		log.Println("Error reading response body:", err)
// 		return
// 	}

// 	responseBody := string(bodyBytes)
// 	value := gjson.Parse(responseBody).Array()
// 	symbols = []string{}
// 	for _, symbol := range value {
// 		symbolCoin := symbol.Get("symbol").String()

// 		if !strings.HasSuffix(symbolCoin, "USDT") {
// 			continue
// 		}
// 		baseAsset := symbolCoin[:len(symbolCoin)-4]
// 		if strings.HasSuffix(baseAsset, "DOWN") || strings.HasSuffix(baseAsset, "UP") {
// 			continue
// 		}
// 		volume24h := symbol.Get("quoteVolume").Float()

// 		if volume24h > 5_000_000 && !slices.Contains(config.Exclude, baseAsset) {
// 			symbols = append(symbols, baseAsset)
// 		}

// 	}
// }
