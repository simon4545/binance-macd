package bn

import (
	"context"
	"fmt"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
	"github.com/markcheno/go-talib"
	"github.com/samber/lo"
	"github.com/simon4545/binance-macd/configuration"
	"github.com/spf13/cast"
)

var client *futures.Client
var Amplitudes = make(map[string]float64)
var ledisdb *ledis.DB
var config *configuration.Config

func InitLedis(ledisdb **ledis.DB) {
	cfg := lediscfg.NewConfigDefault()
	l, _ := ledis.Open(cfg)
	db, _ := l.Select(0)
	*ledisdb = db
}
func Init(fclient *futures.Client, fconfig *configuration.Config) {
	client = fclient
	config = fconfig
	InitLedis(&ledisdb)
	GetSymbolInfo(client)
	GetFeeInfo(client, lo.Keys(config.Symbols))
	InitWS(client)
	// HandleSymbol("BTCUSDT")
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for range ticker.C {
			fmt.Println("定时任务执行，当前时间：", configuration.AtrMap)
		}
	}()
}
func HandleSymbol(k string) {
	// if len(SymbolPrice[k]) > 598 {
	// 	defer functions.TimeTrack(time.Now(), "Handle")
	// }

	// lastPrice := AssetInfo[k].Price
	lastClose, _ := lo.Nth(AssetInfo[k].Close, -1)
	lastClose2, _ := lo.Nth(AssetInfo[k].Close, -2)
	mids := talib.Sma(AssetInfo[k].Close, 20)
	mid, _ := lo.Nth(mids, -1)
	upper := configuration.AtrMap[k]*config.Symbols[k].Multi + lastClose
	lower := lastClose - configuration.AtrMap[k]*config.Symbols[k].Multi

	result, _ := ledisdb.Get([]byte(k))
	hasOrder := BytesToInt(result)
	//做多
	if lastClose > upper && lastClose2 < upper && hasOrder == 0 {
		if checkPosition(k, futures.PositionSideTypeLong) {
			return
		}
		ledisdb.Set([]byte(k), IntToBytes(1))
		placeOrder(k, futures.SideTypeBuy, futures.PositionSideTypeLong)
	}

	//做空
	if lastClose < lower && lastClose2 > lower && hasOrder == 0 {
		if checkPosition(k, futures.PositionSideTypeShort) {
			return
		}
		ledisdb.Set([]byte(k), IntToBytes(1))
		placeOrder(k, futures.SideTypeSell, futures.PositionSideTypeShort)
	}
	//平多
	if lastClose < mid && hasOrder == 1 {
		ledisdb.Set([]byte(k), IntToBytes(0))
		placeOrder(k, futures.SideTypeSell, futures.PositionSideTypeLong)
	}
	//平空
	if lastClose > mid && hasOrder == 1 {
		ledisdb.Set([]byte(k), IntToBytes(0))
		placeOrder(k, futures.SideTypeBuy, futures.PositionSideTypeShort)
	}
	// currentTime := time.Now()
	// formattedTime := []byte(currentTime.Format("2006-01-02"))
}

// 检测当前是否有持仓
func checkPosition(symbol string, positionSide futures.PositionSideType) bool {
	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return false
	}

	for _, position := range positions {
		PositionAmt := cast.ToFloat64(position.PositionAmt)
		if position.Symbol == symbol && PositionAmt != 0 && position.PositionSide == string(positionSide) {
			return true
		}
	}
	return false
}
