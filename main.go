package main

import (
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/remeh/sizedwaitgroup"
	"github.com/tidwall/gjson"
)

var client *binance.Client
var binanceExcludes = []string{"USDC", "TUSD", "USDP", "FDUSD", "AEUR", "ASR", "TFUEL", "OG", "WNXM", "WBETH", "WBTC",
	"WAXP", "FOR", "JST", "SUN", "WIN", "TRX", "UTK", "TROY", "WRX", "DOCK", "C98", "EUR", "USTC", "USDS", "AUD", "DAI", "EPX"}
var symbols []string

func init() {
	lotSizeMap = make(map[string]float64)
	priceFilterMap = make(map[string]float64)
	config = &Config{}
	InitConfig(config)
	client = binance.NewClient(config.BAPI_KEY, config.BAPI_SCRET)
}

func main() {
	GetSymbolInfo(client)
}

func HotList() {
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
		if len(config.SYMBOLS) > 0 && slices.Contains(config.SYMBOLS, baseAsset) {
			symbols = append(symbols, baseAsset)
		}
		if len(config.SYMBOLS) == 0 {
			if volume24h > 5_000_000 && !slices.Contains(binanceExcludes, baseAsset) {
				symbols = append(symbols, baseAsset)
			}
		}
	}
	// sort.Sort(symols)
	// sort.Slice(symbols, func(i, j int) bool {
	// 	return symbols[i].Percent > symbols[j].Percent
	// })
	// symbols = symbols[:100]
	swg := sizedwaitgroup.New(4)

	for _, s := range symbols {
		swg.Add()
		go func(s string) {
			defer swg.Done()
			time.Sleep(time.Millisecond * 100)
		}(s)
	}
	swg.Wait()
}
