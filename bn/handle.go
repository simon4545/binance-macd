package bn

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

var priceUpdateLocker sync.Mutex
var client *futures.Client
var Amplitudes = make(map[string]float64)
var Atrs = make(map[string]float64)
var SymbolPrice = make(map[string][]float64)
var Symbols = []string{"BTCUSDT", "XRPUSDT", "SOLUSDT", "DOGEUSDT"}
var SymbolDebet = map[string]string{"BTCUSDT": "0.003", "XRPUSDT": "100", "SOLUSDT": "1", "DOGEUSDT": "1000"}
var SymbolStepSize = map[string]float64{"BTCUSDT": 0.1, "XRPUSDT": 0.0001, "SOLUSDT": 0.01, "DOGEUSDT": 0.00001}
var LotSizeMap map[string]float64
var PriceFilterMap map[string]float64
var FeeMap map[string]float64
var ledisdb *ledis.DB

func InitLedis(ledisdb **ledis.DB) {
	cfg := lediscfg.NewConfigDefault()
	l, _ := ledis.Open(cfg)
	db, _ := l.Select(0)
	*ledisdb = db
}
func Init(fclient *futures.Client) {
	client = fclient
	InitWS(client)
	InitLedis(&ledisdb)
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for t := range ticker.C {
			// HandleUpdatePrice()
			fmt.Println("定时任务执行，当前时间：", t)
			for _, k := range Symbols {
				prices := SymbolPrice[k]
				lastPrice, _ := lo.Nth(prices, -1)
				fmt.Println(
					k,
					len(prices),
					"最新价格",
					lastPrice,
					"最高价",
					lo.Max(prices),
					"最低价",
					lo.Min(prices),
					strconv.FormatFloat(Atrs[k], 'f', 2, 64))
				// 	HandleSymbol(k)
			}
		}
	}()
}
func HandleSymbol(k string) {
	// if len(SymbolPrice[k]) > 598 {
	// 	defer functions.TimeTrack(time.Now(), "Handle")
	// }
	prices := SymbolPrice[k]
	// prices = lo.Subset(prices, -600, math.MaxInt32)
	max := slices.Max(prices)
	min := slices.Min(prices)
	lastPrice, _ := lo.Nth(prices, -1)
	currentTime := time.Now()
	formattedTime := []byte(currentTime.Format("2006-01-02"))
	result, err := ledisdb.Get(formattedTime)
	if result != nil && cast.ToInt(string(result)) > 3 {
		return
	}

	if lastPrice-min > Atrs[k]*2 {
		fmt.Println(k, "上涨出大事了")
		if !checkPosition(client, k) {
			if err = placeOrder(client, k, futures.SideTypeBuy, futures.PositionSideTypeLong); err == nil {
				ledisdb.Incr(formattedTime)
			}
		}
	} else if max-lastPrice > Atrs[k]*2 {
		fmt.Println(k, "下跌出大事了")
		if !checkPosition(client, k) {
			if err = placeOrder(client, k, futures.SideTypeSell, futures.PositionSideTypeShort); err == nil {
				ledisdb.Incr([]byte(formattedTime))
			}
		}
	}
}
func HandleUpdatePrice() {
	priceUpdateLocker.Lock()
	defer priceUpdateLocker.Unlock()
	for _, k := range Symbols {
		SymbolPrice[k] = lo.Subset(SymbolPrice[k], -600, math.MaxInt32)
		// lenN := len(SymbolPrice[k])
		// if lenN > 150 {
		// 	SymbolPrice[k] = SymbolPrice[k][lenN-150:]
		// }
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
