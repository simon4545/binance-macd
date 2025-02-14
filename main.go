package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
)

const (
	apiKey     = "xzJKM9OUwYXxVrOpG9474d2Tgqx57QyABMzIekxXXzDSRNN5ClsNYlDblVVDqaNx"
	secretKey  = "NG7W8uzFSu3PGnIx3lAyxIU232rhrQGsIz8n124A5eIlGeKHRnxKNji3V1cLgyzf"
	symbol     = "ETHUSDT"
	interval   = "5m"
	rsiPeriod  = 14
	limit      = 300
	takeProfit = 0.004 // 千分之四
	stopLoss   = 0.002 // 千分之二
	quantity   = 0.05  // 交易数量

)

var (
	client       *futures.Client
	mu           sync.Mutex
	rsiValues    []float64
	closes       []float64
	entryPrice   float64
	positionOpen bool
	orderID      int64
	wsStop       chan struct{}
	doneC        chan struct{}
)

func main() {
	closes = make([]float64, 0)
	// 初始化币安客户端
	client = futures.NewClient(apiKey, secretKey)

	// 获取历史K线数据以初始化RSI
	initRSI()

	// 启动WebSocket订阅K线数据
	go subscribeKlineWebSocket()
	go wsUserReConnect()
	go PrintRSI()
	// 保持主程序运行
	select {}
}
func PrintRSI() {
	for {
		fmt.Println(rsiValues[len(rsiValues)-1], closes[len(closes)-1])
		time.Sleep(time.Second * 10)
	}
}

// 初始化RSI值
func initRSI() {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get historical klines: %v", err)
	}

	for _, kline := range klines {
		closes = append(closes, parseFloat(kline.Close))
	}

	rsiValues = talib.Rsi(closes[len(closes)-61:], rsiPeriod)
	rsiValues = rsiValues[15:]
	log.Printf("Initialized RSI with %d values\n", len(rsiValues))
}

// 订阅K线WebSocket
func subscribeKlineWebSocket() {
	wsKlineHandler := func(event *binance.WsKlineEvent) {
		closePrice := parseFloat(event.Kline.Close)
		// 只在K线结束时处理
		if event.Kline.IsFinal {
			closes[len(closes)-1] = closePrice
			closes = append(closes, closePrice)
		} else {
			closes[len(closes)-1] = closePrice
		}

		updateRSI(closePrice)
		checkTradingSignal(closePrice)

	}

	errHandler := func(err error) {
		log.Printf("WebSocket error: %v", err)
	}
	var err error
	doneC, wsStop, err = binance.WsKlineServe(symbol, interval, wsKlineHandler, errHandler)
	if err != nil {
		log.Fatalf("Failed to start WebSocket: %v", err)
	}
	// defer stopC()

	log.Println("WebSocket connected, waiting for kline data...")

	// 保持WebSocket连接
	<-doneC
}
func wsUserReConnect() {
	for {
		time.Sleep(55 * time.Minute)
		fmt.Println("1hour reconnect wsUser")
		wsStop <- struct{}{}
	}
}

// 更新RSI值
func updateRSI(closePrice float64) {
	mu.Lock()
	defer mu.Unlock()

	// // 添加最新价格到历史数据
	// if len(rsiValues) >= limit {
	// 	rsiValues = rsiValues[1:]
	// }
	// rsiValues = append(rsiValues, closePrice)

	// 计算RSI
	// closes := make([]float64, len(rsiValues))
	// for i, price := range rsiValues {
	// 	closes[i] = price
	// }
	rsi := talib.Rsi(closes[len(closes)-61:], rsiPeriod)
	rsi = rsi[15:]
	// 更新RSI值
	rsiValues = rsi
}

// 检查交易信号
func checkTradingSignal(currentPrice float64) {
	mu.Lock()
	defer mu.Unlock()

	// 找到最高的5个RSI值
	top5RSI := getTop4RSI(rsiValues)

	// 计算最高5个RSI的平均值
	averageTop5RSI := calculateAverage(top5RSI)

	// 获取当前RSI值
	currentRSI := rsiValues[len(rsiValues)-1]

	// 判断是否做空
	if currentRSI > averageTop5RSI && !positionOpen {
		log.Println("当前RSI高于最高5个RSI的平均值，执行做空操作")
		entryPrice = currentPrice
		positionOpen = true

		// 执行做空操作
		order, err := createOrder("SELL", quantity)
		if err != nil {
			log.Printf("Failed to create order: %v", err)
			return
		}
		orderID = order.OrderID
		log.Printf("做空订单已创建，订单ID: %d, 成交价格: %f\n", orderID, entryPrice)

		// 启动止盈止损监控
		go monitorTakeProfitStopLoss(currentPrice)
	}
}

// 创建订单
func createOrder(side string, quantity float64) (*futures.CreateOrderResponse, error) {
	order, err := client.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideType(side)).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%f", quantity)).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
	return order, nil
}

// 监控止盈止损
func monitorTakeProfitStopLoss(entryPrice float64) {
	for {
		currentPrice := closes[len(closes)-1]
		// 计算盈亏比例
		profit := (entryPrice - currentPrice) / entryPrice

		if profit >= takeProfit {
			log.Println("达到止盈条件，平仓")
			closePosition(currentPrice)
			goto EXIT
		} else if profit <= -stopLoss {
			log.Println("达到止损条件，平仓")
			closePosition(currentPrice)
			goto EXIT
		}
		time.Sleep(time.Millisecond * 500)
	}
EXIT:
	return
}

// 平仓
func closePosition(exitPrice float64) {
	mu.Lock()
	defer mu.Unlock()

	// 执行平仓操作
	order, err := createOrder("BUY", quantity)
	if err != nil {
		log.Printf("Failed to close position: %v", err)
		return
	}
	log.Printf("平仓订单已创建，订单ID: %d, 成交价格: %f\n", order.OrderID, exitPrice)
	positionOpen = false
	time.Sleep(time.Minute * 10)
}

// 找到最高的5个RSI值
func getTop4RSI(rsiValues []float64) []float64 {
	copied := make([]float64, len(rsiValues))
	copy(copied, rsiValues)
	sort.Float64s(copied)
	return copied[len(copied)-4 : len(copied)-1]
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
