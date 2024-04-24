package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/remeh/sizedwaitgroup"
	"github.com/tidwall/gjson"
)

var client *futures.Client
var symbols []string
var atrMap map[string]float64

func check() {
	currentTime := time.Now()
	targetYear := time.Date(2024, time.June, 1, 0, 0, 0, 0, currentTime.Location())
	if currentTime.After(targetYear) {
		os.Exit(0) // 使用非零状态码退出，表示程序是有意退出的
	}
}
func init() {
	check()

	lotSizeMap = make(map[string]float64)
	priceFilterMap = make(map[string]float64)
	atrMap = make(map[string]float64)
	feeMap = make(map[string]float64)

	config = &Config{}
	InitConfig(config)
	InitDB()
	client = binance.NewFuturesClient(config.BAPI_KEY, config.BAPI_SCRET)
}

func checkCross(client *futures.Client, symbol string) {
	// defer time.Sleep(4 * time.Second)
	klines, err := client.NewKlinesService().Symbol(symbol + "USDT").Interval(config.Period).Limit(200).Do(context.Background())
	if err != nil {
		print(err)
		return
	}
	closingPrices := []float64{}
	highPrices := []float64{}
	lowPrices := []float64{}
	for _, kline := range klines {
		close, _ := strconv.ParseFloat(kline.Close, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		closingPrices = append(closingPrices, close)
		highPrices = append(highPrices, high)
		lowPrices = append(lowPrices, low)
	}
	lastPrice, _ := strconv.ParseFloat(klines[len(klines)-1].Close, 64)
	Handle(config, symbol, lastPrice, closingPrices, highPrices, lowPrices)
}

func CheckCross(client *futures.Client) {
	for {
		fmt.Println(time.Now(), "开启新的一启")
		swg := sizedwaitgroup.New(4)
		for _, s := range symbols {
			swg.Add()
			go func(s string) {
				defer swg.Done()
				pair := fmt.Sprintf("%sUSDT", s)
				if lotSizeMap[pair] != 0 {
					checkCross(client, s)
					time.Sleep(time.Millisecond * 100)
				} else {
					fmt.Println("交易对", pair, "不可交易")
					return
				}
			}(s)
		}
		swg.Wait()
		time.Sleep(time.Second * 10)
	}
}
func main() {
	InitWS()
	GetSymbolInfo(client)
	go CheckAtr(client)
	go CheckCross(client)
	select {}
}

func list() {
	if len(config.Symbols) > 0 {
		symbols = []string{}
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
		symbols = []string{}
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

			if volume24h > 5_000_000 && !slices.Contains(config.Exclude, baseAsset) {
				symbols = append(symbols, baseAsset)
			}

		}
	}
	// sort.Sort(symols)
	// sort.Slice(symbols, func(i, j int) bool {
	// 	return symbols[i].Percent > symbols[j].Percent
	// })
	// symbols = symbols[:100]
	fmt.Println("总大小", len(symbols))

}
