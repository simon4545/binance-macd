package bn

import (
	"context"
	"fmt"
	"slices"
	"strconv"
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
var SymbolStepSize = make(map[string]float64)
var LotSizeMap map[string]float64
var PriceFilterMap map[string]float64
var FeeMap map[string]float64

func Init(fclient *futures.Client) {
	client = fclient
	InitWS(client)
	SymbolDebet["BTCUSDT"] = "0.003"
	SymbolDebet["XRPUSDT"] = "100"
	SymbolDebet["SOLUSDT"] = "1"
	SymbolDebet["DOGEUSDT"] = "1000"
	SymbolStepSize["BTCUSDT"] = 0.1
	SymbolStepSize["XRPUSDT"] = 0.0001
	SymbolStepSize["SOLUSDT"] = 0.01
	SymbolStepSize["DOGEUSDT"] = 0.00001
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
				fmt.Println(
					k,
					len(prices),
					"最新价格",
					strconv.FormatFloat(lastPrice, 'f', 2, 64), k,
					"最高价",
					strconv.FormatFloat(max, 'f', 2, 64),
					"最低价",
					strconv.FormatFloat(min, 'f', 2, 64),
					strconv.FormatFloat(Atrs[k], 'f', 2, 64))
				if lastPrice-min > Atrs[k]*2 {
					fmt.Println(k, "上涨出大事了")
					if checkPosition(client, k) == false {
						placeOrder(client, k, futures.SideTypeBuy, futures.PositionSideTypeLong)
					}
				} else if max-lastPrice > Atrs[k]*2 {
					fmt.Println(k, "下跌出大事了")
					if checkPosition(client, k) == false {
						placeOrder(client, k, futures.SideTypeSell, futures.PositionSideTypeShort)
					}
				}
			}

		}
	}()
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

// 检测当前是否有持仓
func checkPosition(client *futures.Client, symbol string) bool {
	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return false
	}

	for _, position := range positions {
		PositionAmt := cast.ToFloat64(position.PositionAmt)
		if position.Symbol == symbol && PositionAmt != 0 {
			return true
		}
	}
	return false
}
