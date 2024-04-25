package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
	"github.com/remeh/sizedwaitgroup"

	"github.com/simon4545/binance-macd/config"
	"github.com/simon4545/binance-macd/db"
	_ "github.com/simon4545/binance-macd/execute/long"
	_ "github.com/simon4545/binance-macd/execute/short"
	"github.com/simon4545/binance-macd/interfacer"
	"github.com/simon4545/binance-macd/utils"
)

var conf *config.Config
var client *futures.Client
var symbols = []string{"BTC", "ETH"}

func check() {
	currentTime := time.Now()
	targetYear := time.Date(2024, time.June, 1, 0, 0, 0, 0, currentTime.Location())
	if currentTime.After(targetYear) {
		os.Exit(0) // 使用非零状态码退出，表示程序是有意退出的
	}
}
func init() {
	check()

	config.LotSizeMap = make(map[string]float64)
	config.PriceFilterMap = make(map[string]float64)
	config.AtrMap = make(map[string]float64)
	config.FeeMap = make(map[string]float64)

	conf = &config.Config{}
	config.InitConfig(conf)

	db.InitDB()

	client = binance.NewFuturesClient(conf.BAPI_KEY, conf.BAPI_SCRET)
}

func checkCross(client *futures.Client, symbol string) {
	// defer time.Sleep(4 * time.Second)
	klines, err := client.NewKlinesService().Symbol(symbol + "USDT").Interval(conf.Period).Limit(200).Do(context.Background())
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
	excutor := interfacer.Create(conf.Side, client)
	excutor.Handle(client, conf, symbol, lastPrice, closingPrices, highPrices, lowPrices)
}

func CheckCross(client *futures.Client, symbols []string) {
	for {
		// fmt.Println(time.Now(), "开启新的一启")
		swg := sizedwaitgroup.New(4)
		for _, s := range symbols {
			swg.Add()
			go func(s string) {
				defer swg.Done()
				pair := fmt.Sprintf("%sUSDT", s)
				if config.LotSizeMap[pair] != 0 {
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

func checkAtr(client *futures.Client, symbol string) {
	pair := fmt.Sprintf("%sUSDT", symbol)
	if config.LotSizeMap[pair] == 0 {
		fmt.Println("交易对", pair, "不可交易")
		return
	}
	klines, err := client.NewKlinesService().Symbol(pair).Interval("1h").Limit(100).Do(context.Background())
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
	atr := talib.Atr(highPrices, lowPrices, closingPrices, 12)
	config.AtrMap[symbol] = atr[len(atr)-1]
	// fmt.Println(symbol, "atr", atrMap[symbol])
}
func CheckAtr(client *futures.Client, symbols []string) {
	for {
		// fmt.Println(time.Now(), "开启新的一启")
		swg := sizedwaitgroup.New(4)
		for _, s := range symbols {
			swg.Add()
			go func(s string) {
				defer swg.Done()
				checkAtr(client, s)
				time.Sleep(time.Millisecond * 100)
			}(s)
		}
		swg.Wait()
		time.Sleep(time.Second * 20)
	}
}
func main() {
	InitWS()
	utils.List(conf, &symbols)
	utils.GetSymbolInfo(client, symbols)
	go CheckAtr(client, symbols)
	go CheckCross(client, symbols)
	select {}
}
