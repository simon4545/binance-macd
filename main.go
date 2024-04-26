package main

import (
	"context"
	"fmt"
	"net/http"
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
	"github.com/simon4545/binance-macd/tools"
	"github.com/simon4545/binance-macd/utils"
)

var conf *config.Config
var client *futures.Client
var symbols = []string{"BTCUSDT", "ETHUSDT"}

func check() {
	currentTime := time.Now()
	targetYear := time.Date(2024, time.June, 1, 0, 0, 0, 0, currentTime.Location())
	if currentTime.After(targetYear) {
		os.Exit(0) // 使用非零状态码退出，表示程序是有意退出的
	}
}

// transport binance transport client
type Transport struct {
	UnderlyingTransport http.RoundTripper
}

var weightMaxPerMinute int = 6000
var usedWeight int

// RoundTrip implement http roundtrip
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.UnderlyingTransport.RoundTrip(req)
	if resp != nil && resp.Header != nil {
		usedWeight, _ = strconv.Atoi(resp.Header.Get("X-Mbx-Used-Weight-1m"))
		if usedWeight > weightMaxPerMinute/2 {
			fmt.Println("请求权重", usedWeight)
			time.Sleep(time.Second * 30)
		}
	}

	return resp, err
}
func init() {
	check()

	config.LotSizeMap = make(map[string]float64)
	config.PriceFilterMap = make(map[string]float64)
	config.AtrMap = tools.NewSafeMap[string, float64]()
	config.FeeMap = make(map[string]float64)

	conf = &config.Config{}
	config.InitConfig(conf)

	db.InitDB()

	client = binance.NewFuturesClient(conf.BAPI_KEY, conf.BAPI_SCRET)
	client.NewSetServerTimeService().Do(context.Background())
	c := http.Client{Transport: &Transport{UnderlyingTransport: http.DefaultTransport}}
	client.HTTPClient = &c
	binance.WebsocketKeepalive = true
}

func checkCross(client *futures.Client, symbol string) {
	// defer time.Sleep(4 * time.Second)
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(conf.Period).Limit(100).Do(context.Background())
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
	side := conf.Symbols[symbol].Side
	if side == "BOTH" {
		excutor := interfacer.Create("LONG", client)
		excutor.Handle(client, conf, symbol, lastPrice, closingPrices, highPrices, lowPrices)
		time.Sleep(time.Second * 2)
		excutors := interfacer.Create("SHORT", client)
		excutors.Handle(client, conf, symbol, lastPrice, closingPrices, highPrices, lowPrices)
	} else {
		excutor := interfacer.Create(side, client)
		excutor.Handle(client, conf, symbol, lastPrice, closingPrices, highPrices, lowPrices)
	}

}

func CheckCross(client *futures.Client) {
	for {
		utils.List(conf, &symbols)
		// fmt.Println(time.Now(), "开启新的一启")
		swg := sizedwaitgroup.New(4)
		for _, s := range symbols {
			swg.Add()
			go func(pair string) {
				defer swg.Done()
				if config.LotSizeMap[pair] != 0 {
					checkCross(client, pair)
					time.Sleep(time.Millisecond * 100)
				} else {
					fmt.Println("交易对", pair, "不可交易")
					return
				}
			}(s)
		}
		swg.Wait()
		time.Sleep(time.Second * 6)
	}
}

func checkAtr(client *futures.Client, symbol string) {
	pair := symbol
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
	config.AtrMap.Set(symbol, atr[len(atr)-1])
	// fmt.Println(symbol, "atr", atrMap[symbol])
}
func CheckAtr(client *futures.Client) {
	for {
		utils.GetFeeInfo(client, symbols)
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
	utils.GetSymbolInfo(client)
	go CheckAtr(client)
	go CheckCross(client)
	select {}
}
