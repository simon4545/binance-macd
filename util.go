package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
)

func PrintRSI() {
	for {
		time.Sleep(time.Second * 10)
		fmt.Println(rsiValues[len(rsiValues)-1], currentPrice, averageTop5RSI, averageBottom5RSI, lastAtr/currentPrice)
	}
}

// 初始化RSI值
func initRSI() {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get historical klines: %v", err)
	}

	for _, kline := range klines {
		if kline.CloseTime > time.Now().UnixMilli() {
			continue
		}
		closes = append(closes, parseFloat(kline.Close))
		highs = append(highs, parseFloat(kline.High))
		lows = append(lows, parseFloat(kline.Low))
	}
	atr := talib.Atr(highs, lows, closes, 12)
	lastAtr = atr[len(atr)-1]
	rsiValues = talib.Rsi(closes[len(closes)-91:], rsiPeriod)
	rsiValues = rsiValues[len(rsiValues)-48:]
	currentPrice = parseFloat(klines[len(klines)-1].Close)
	log.Printf("Initialized RSI with %d values\n", len(rsiValues))
}

// 更新RSI值
func updateRSI() {
	mu.Lock()
	defer mu.Unlock()
	rsi := talib.Rsi(closes[len(closes)-91:], rsiPeriod)
	rsi = rsi[len(rsi)-48:]
	// 更新RSI值
	rsiValues = rsi
	atr := talib.Atr(highs, lows, closes, 7)
	lastAtr = atr[len(atr)-1]
}

// 找到最高的5个RSI值
func getTop4RSI(rsiValues []float64) []float64 {
	copied := make([]float64, len(rsiValues))
	copy(copied, rsiValues)
	sort.Float64s(copied)
	return copied[len(copied)-6 : len(copied)-1]
}

// 找到最高的5个RSI值
func getBottom4RSI(rsiValues []float64) []float64 {
	copied := make([]float64, len(rsiValues))
	copy(copied, rsiValues)
	sort.Float64s(copied)
	return copied[1:6]
}

// 计算平均值
func calculateAverage(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// 将字符串转换为float64
func parseFloat(s string) float64 {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		log.Fatalf("Failed to parse float: %v", err)
	}
	return f
}

// 创建订单
func createOrder(positionside futures.PositionSideType, side string, quantity float64) (*futures.CreateOrderResponse, error) {
	orderPre := client.NewCreateOrderService().
		Symbol(symbol).
		PositionSide(positionside).
		Side(futures.SideType(side)).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%f", quantity))
	order, err := orderPre.Do(context.Background())
	if err != nil {
		return nil, err
	}
	return order, nil
}

// 平仓
func openPosition(positionside futures.PositionSideType, quantity float64) (*futures.CreateOrderResponse, error) {
	var side string
	if positionside == futures.PositionSideTypeShort {
		side = "SELL"
	} else {
		side = "BUY"
	}
	// 执行开仓操作
	order, err := createOrder(positionside, side, quantity)
	if err != nil {
		log.Printf("Failed to close position: %v", err)
		return nil, err
	}
	entryPrice = currentPrice
	positionOpen = true
	positionATR = lastAtr
	positionTime = time.Now()
	return order, nil
	// time.Sleep(time.Minute * 10)
}

// 平仓
func closePosition(positionside futures.PositionSideType, loss bool) {
	var side string
	if positionside == futures.PositionSideTypeShort {
		side = "BUY"
	} else {
		side = "SELL"
	}
	// 执行平仓操作
	order, err := createOrder(positionside, side, quantity)
	if err != nil {
		log.Printf("Failed to close position: %v", err)
		return
	}
	log.Printf("平仓订单已创建，订单ID: %d, 成交价格: %f\n", order.OrderID, currentPrice)
	positionOpen = false
	positionTime = time.Now().Add(time.Hour * 1000000)
	if loss {
		protectTime = time.Now().Add(time.Minute * 10)
	}
	// time.Sleep(time.Minute * 10)
}
