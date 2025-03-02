package bn

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/spf13/cast"
)

type KLine struct {
	Price float64
	Date  []int64
	Open  []float64
	Close []float64
	High  []float64
	Low   []float64
}

func getListKlines(pair string) {
	klines, err := client.NewKlinesService().Symbol(pair).Interval("4h").Limit(288).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(pair)
	if AssetInfo[pair] == nil {
		AssetInfo[pair] = &KLine{
			Date:  []int64{},
			Open:  []float64{},
			Close: []float64{},
			High:  []float64{},
			Low:   []float64{},
		}
	}
	assetInfo := AssetInfo[pair]
	assetInfo.Price, _ = strconv.ParseFloat(klines[len(klines)-1].Close, 64)
	for _, k := range klines {
		if k.CloseTime > time.Now().UnixMilli() {
			continue
		}
		// tm := time.Unix(k.OpenTime, 0).Local()
		timestamp := k.OpenTime
		// unixTime := timestamp / 1000 // 将时间戳转为秒级时间戳
		// nanoSecond := (timestamp % 1000000) * 1000 // 将时间戳的微秒部分转为纳秒
		// timeFromUnix := time.Unix(unixTime, 0).Local()
		close, _ := strconv.ParseFloat(k.Close, 64)
		open, _ := strconv.ParseFloat(k.Open, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		assetInfo.Close = append(assetInfo.Close, close)
		assetInfo.Open = append(assetInfo.Open, open)
		assetInfo.High = append(assetInfo.High, high)
		assetInfo.Low = append(assetInfo.Low, low)
		assetInfo.Date = append(assetInfo.Date, timestamp)
	}

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
func WsTicker() {
	for {
		var err error
		var doneC chan struct{}

		errHandler := func(err error) {
			log.Printf("Error: %v", err)
		}
		wsKlineHandler := func(event *futures.WsMarkPriceEvent) {
			if slices.Contains(Symbols, event.Symbol) {
				SymbolPrice[event.Symbol] = append(SymbolPrice[event.Symbol], cast.ToFloat64(event.MarkPrice))
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
func WsKline() {
	symbolsWithInterval := make(map[string]string)
	for _, k := range Symbols {
		symbolsWithInterval[k] = "5m"
	}
	for {
		var err error
		var doneC chan struct{}

		errHandler := func(err error) {
			log.Printf("Error: %v", err)
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
