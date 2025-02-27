package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

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
	// entryPrice = currentPrice
	// positionOpen = true
	// positionATR = 0.002
	// positionTime = time.Now()
	return order, nil
	// time.Sleep(time.Minute * 10)
}

// 平仓
func closePosition(positionside futures.PositionSideType, quantity float64, loss bool) {
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
	// positionOpen = false
	// positionTime = time.Now().Add(time.Hour * 1000000)
	if loss {
		protectTime = time.Now().Add(time.Hour * 10)
	}
	// time.Sleep(time.Minute * 10)
}
