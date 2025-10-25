package bn

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/simon4545/binance-macd/configuration"
)

func getListKlines(pair string) {
	klines, err := client.NewKlinesService().Symbol(pair).Interval(c.Symbols[pair].Period).Limit(288).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(pair)
	if AssetInfo[pair] == nil {
		AssetInfo[pair] = &configuration.KLine{
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
func wsKlineHandler(event *binance.WsKlineEvent) {
	k := event.Kline
	assetInfo := AssetInfo[k.Symbol]
	timestamp := k.StartTime
	// unixTime := timestamp / 1000 // 将时间戳转为秒级时间戳
	// timeFromUnix := time.Unix(unixTime, 0).Local()

	close, _ := strconv.ParseFloat(k.Close, 64)
	open, _ := strconv.ParseFloat(k.Open, 64)
	high, _ := strconv.ParseFloat(k.High, 64)
	low, _ := strconv.ParseFloat(k.Low, 64)
	assetInfo.Price = close
	// volume, _ := strconv.ParseInt(k.Volume, 10, 64)
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
	if k.IsFinal {
		fmt.Println("lastKline", lastKline, timestamp, "changdu", len(assetInfo.Close), k.Symbol)
	}
}

// websocket K线
func WsKline() {
	symbolsWithInterval := make(map[string]string)
	for k, symbol := range c.Symbols {
		symbolsWithInterval[k] = symbol.Period
	}
	for {
		var err error
		var doneC chan struct{}

		errHandler := func(err error) {
			log.Printf("Error: %v", err)
		}

		// 启动 WebSocket K 线监听
		doneC, _, err = binance.WsCombinedKlineServe(symbolsWithInterval, wsKlineHandler, errHandler)
		if err != nil {
			log.Printf("Failed to start WebSocket for %s: %v", symbolsWithInterval, err)
			time.Sleep(3 * time.Second)
			continue
		}
		<-doneC
		time.Sleep(3 * time.Second)
	}
}
