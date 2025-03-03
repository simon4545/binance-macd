package bn

import (
	"context"
	"fmt"
	"log"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/configuration"
	"github.com/simon4545/binance-macd/functions"
	"github.com/spf13/cast"
)

var AssetInfo map[string]*KLine
var wsUserStop chan struct{}

func InitWS(client *futures.Client) {
	AssetInfo = make(map[string]*KLine)
	go CheckATR()
	// go wsUser(client)
	// go wsUserReConnect()
	// go WsTicker(HandleSymbol)
	go WsKline(HandleSymbol)
}

func getUserStream(client *futures.Client) string {
	res, err := client.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return res
}

func wsUser(client *futures.Client) {
	listenKey := getUserStream(client)
	errHandler := func(err error) {
		fmt.Println("ws user error:", err)
	}
	var err error
	var doneC chan struct{}
	doneC, wsUserStop, err = binance.WsUserDataServe(listenKey, userWsHandler, errHandler)
	if err != nil {
		fmt.Println("ws user error:", err)
		return
	}
	<-doneC
	time.Sleep(3 * time.Second)
	wsUser(client)
}

func wsUserReConnect() {
	for {
		time.Sleep(55 * time.Minute)
		fmt.Println("1hour reconnect wsUser")
		wsUserStop <- struct{}{}
	}
}

func userWsHandler(event *binance.WsUserDataEvent) {
	if event.Event != "executionReport" {
		return
	}
	message := event.OrderUpdate
	if !strings.HasSuffix(message.Symbol, "USDT") {
		return
	}

	// if message.Status == "CANCELED" {

	// }
	if message.Status == "FILLED" {
		price, _ := strconv.ParseFloat(message.LatestPrice, 64)
		symbol := message.Symbol[:len(message.Symbol)-4]
		feeCost, _ := decimal.NewFromString(message.FeeCost)
		filledVolume, _ := decimal.NewFromString(message.FilledVolume)
		gainVolume := filledVolume.Mul(decimal.NewFromFloat(1 - configuration.FeeMap[message.Symbol])).InexactFloat64()
		step := decimal.NewFromFloat(configuration.LotSizeMap[message.Symbol])
		gainVolume = functions.RoundStepSizeDecimal(gainVolume, step.InexactFloat64()).InexactFloat64()
		quoteVolume, _ := strconv.ParseFloat(message.FilledQuoteVolume, 64)
		fmt.Println("订单成交", symbol, quoteVolume, gainVolume, price, feeCost.InexactFloat64())
		// quantity, _ := strconv.ParseFloat(message.Volume, 64)
		if strings.HasPrefix(message.ClientOrderId, "SIM-") && message.Side == string(binance.SideTypeBuy) {
			// fmt.Println("订单成交-量化", symbol, quoteVolume, gainVolume, price)
			// invest := Investment{
			// 	Operate:   "BUY",
			// 	Currency:  symbol,
			// 	Amount:    quoteVolume,
			// 	Quantity:  gainVolume,
			// 	UnitPrice: price,
			// }
			// MakeDBInvestment(invest)
		}

	}
}

type KLine struct {
	Price float64
	Date  []int64
	Open  []float64
	Close []float64
	High  []float64
	Low   []float64
}

func wsKlineHandler(event *futures.WsKlineEvent) {
	k := event.Kline
	assetInfo := AssetInfo[k.Symbol]
	timestamp := k.StartTime

	close, _ := strconv.ParseFloat(k.Close, 64)
	open, _ := strconv.ParseFloat(k.Open, 64)
	high, _ := strconv.ParseFloat(k.High, 64)
	low, _ := strconv.ParseFloat(k.Low, 64)
	assetInfo.Price = close
	// volume, _ := strconv.ParseInt(k.Volume, 10, 64)

	if k.IsFinal {
		lastKline := assetInfo.Date[len(assetInfo.Date)-1]
		if lastKline == timestamp {
			assetInfo.Close[len(assetInfo.Close)-1] = close
			assetInfo.Open[len(assetInfo.Open)-1] = open
			assetInfo.High[len(assetInfo.High)-1] = high
			assetInfo.Low[len(assetInfo.Low)-1] = low
		} else {
			assetInfo.Close = append(assetInfo.Close, close)
			assetInfo.Open = append(assetInfo.Open, open)
			assetInfo.High = append(assetInfo.High, high)
			assetInfo.Low = append(assetInfo.Low, low)
			assetInfo.Date = append(assetInfo.Date, timestamp)
		}
		fmt.Println("lastKline", lastKline, timestamp, "changdu", len(assetInfo.Close), k.Symbol)
	}
}

// websocket Ticker
func WsTicker(callback func(string)) {
	for {
		var err error
		var doneC chan struct{}

		errHandler := func(err error) {
			log.Printf("Error: %v", err)
		}
		wsKlineHandler := func(event *futures.WsMarkPriceEvent) {
			if slices.Contains(Symbols, event.Symbol) {
				SymbolPrice[event.Symbol] = append(SymbolPrice[event.Symbol], cast.ToFloat64(event.MarkPrice))
				callback(event.Symbol)
			}
		}
		doneC, _, err = futures.WsCombinedMarkPriceServe(Symbols, wsKlineHandler, errHandler)
		if err != nil {
			log.Printf("Failed to start WebSocket for: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		<-doneC
		time.Sleep(3 * time.Second)
	}
}

// websocket K线
func WsKline(callback func(string)) {
	symbolsWithInterval := make(map[string]string)
	for _, k := range Symbols {
		symbolsWithInterval[k] = "1m"
	}
	for {
		var err error
		var doneC chan struct{}

		errHandler := func(err error) {
			log.Printf("Error: %v", err)
		}
		wsKlineHandler := func(event *futures.WsKlineEvent) {
			if slices.Contains(Symbols, event.Symbol) {
				k := event.Kline
				close := cast.ToFloat64(k.Close)
				SymbolPrice[event.Symbol] = append(SymbolPrice[event.Symbol], close)
				SymbolPrice[event.Symbol] = lo.Subset(SymbolPrice[event.Symbol], -600, math.MaxInt32)
				callback(event.Symbol)
			}
		}
		// 启动 WebSocket K 线监听
		doneC, _, err = futures.WsCombinedKlineServe(symbolsWithInterval, wsKlineHandler, errHandler)
		if err != nil {
			log.Printf("Failed to start WebSocket for %s: %v", symbolsWithInterval, err)
			time.Sleep(3 * time.Second)
			continue
		}
		<-doneC
		time.Sleep(3 * time.Second)
	}
}
