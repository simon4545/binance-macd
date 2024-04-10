package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/adshao/go-binance/v2"
	"github.com/tidwall/gjson"
)

var client *binance.Client
var binanceExcludes = []string{"USDC", "TUSD", "USDP", "FDUSD", "AEUR", "ASR", "TFUEL", "OG", "WNXM", "WBETH", "WBTC", "STORJ", "SKL", "COMP",
	"WAXP", "FOR", "JST", "SUN", "WIN", "TRX", "UTK", "TROY", "WRX", "DOCK", "C98", "EUR", "USTC", "USDS", "AUD", "DAI", "EPX"}
var symbols []string

func init() {
	lotSizeMap = make(map[string]float64)
	priceFilterMap = make(map[string]float64)
	config = &Config{}
	InitConfig(config)
	InitDB()
	client = binance.NewClient(config.BAPI_KEY, config.BAPI_SCRET)
}

func main() {
	GetSymbolInfo(client)
	go HotList()
	// go CheckCross(client)
	select {}
}

func HotList() {
	symbols = []string{}
	if len(config.Symbols) > 0 {
		symbols = append(symbols, config.Symbols...)
	} else {
		url := "https://api.binance.com/api/v3/ticker/24hr"
		// url := "https://api.binance.com/api/v3/ticker/24hr"
		response, err := http.Get(url)
		if err != nil {
			log.Println("Error making GET request:", err)
			return
		}
		defer response.Body.Close()
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			return
		}

		responseBody := string(bodyBytes)
		value := gjson.Parse(responseBody).Array()
		for _, symbol := range value {
			symbolCoin := symbol.Get("symbol").String()

			if !strings.HasSuffix(symbolCoin, "USDT") {
				continue
			}
			baseAsset := symbolCoin[:len(symbolCoin)-4]
			if strings.HasSuffix(baseAsset, "DOWN") || strings.HasSuffix(baseAsset, "UP") {
				continue
			}
			volume24h := symbol.Get("quoteVolume").Float()
			if len(config.Symbols) > 0 && slices.Contains(config.Symbols, baseAsset) {
				symbols = append(symbols, baseAsset)
			}
			if len(config.Symbols) == 0 {
				if volume24h > 5_000_000 && !slices.Contains(binanceExcludes, baseAsset) {
					symbols = append(symbols, baseAsset)
				}
			}
		}
	}
	// sort.Sort(symols)
	// sort.Slice(symbols, func(i, j int) bool {
	// 	return symbols[i].Percent > symbols[j].Percent
	// })
	// symbols = symbols[:100]
	fmt.Println("总大小", len(symbols))
	CheckCross(client)
}
