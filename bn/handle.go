package bn

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/spf13/cast"
)

var priceUpdateLocker sync.Mutex
var client *futures.Client
var Amplitudes = make(map[string]float64)
var Atrs = make(map[string]float64)
var SymbolPrice = make(map[string][]float64)
var Symbols = []string{"BTCUSDT", "XRPUSDT", "SOLUSDT", "DOGEUSDT"}
var SymbolDebet = make(map[string]string)

func Init(fclient *futures.Client) {
	client = fclient
	InitWS(client)
	SymbolDebet["BTCUSDT"] = "0.003"
	SymbolDebet["XRPUSDT"] = "100"
	SymbolDebet["SOLUSDT"] = "1"
	SymbolDebet["DOGEUSDT"] = "1000"
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for t := range ticker.C {
			HandleUpdatePrice()
			fmt.Println("定时任务执行，当前时间：", t)
			for _, k := range Symbols {
				prices := SymbolPrice[k]
				if len(prices) > 100 {
					prices = prices[len(prices)-100:]
				}
				max := slices.Max(prices)
				min := slices.Min(prices)
				lastPrice := prices[len(prices)-1]
				fmt.Println("ss", len(prices), k, max, min, Atrs[k])
				if lastPrice-min > Atrs[k]*2.5 {
					fmt.Println(k, "上涨出大事了")
				} else if max-lastPrice > Atrs[k]*2.5 {
					fmt.Println(k, "下跌出大事了")
				}
			}
			// for _, k := range Symbols {
			// 	assetInfo := AssetInfo[k]
			// 	Handle(k, assetInfo)
			// }
		}
	}()
	// go functions.CheckCross(client, config.Symbols, config, Handle)
}
func HandleUpdatePrice() {
	priceUpdateLocker.Lock()
	defer priceUpdateLocker.Unlock()
	for _, k := range Symbols {
		lenN := len(SymbolPrice[k])
		if lenN > 150 {
			SymbolPrice[k] = SymbolPrice[k][lenN-100:]
		}
	}
}
func CheckPosition(client *futures.Client, symbol string) {
	// 1. 检测当前是否有持仓
	hasPosition, err := checkPosition(client, symbol)
	if err != nil {
		log.Printf("Error checking position: %v\n", err)
		return
	}
	if hasPosition {
		fmt.Println("Already have a position, skipping...")
		return
	}
}

// 检测当前是否有持仓
func checkPosition(client *futures.Client, symbol string) (bool, error) {
	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return false, err
	}

	for _, position := range positions {
		PositionAmt := cast.ToFloat64(position.PositionAmt)
		if position.Symbol == symbol && PositionAmt != 0 {
			return true, nil
		}
	}
	return false, nil
}
