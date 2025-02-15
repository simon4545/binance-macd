package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

const (
	apiKey     = "xzJKM9OUwYXxVrOpG9474d2Tgqx57QyABMzIekxXXzDSRNN5ClsNYlDblVVDqaNx"
	secretKey  = "NG7W8uzFSu3PGnIx3lAyxIU232rhrQGsIz8n124A5eIlGeKHRnxKNji3V1cLgyzf"
	symbol     = "JUPUSDT"
	interval   = "5m"
	rsiPeriod  = 14
	limit      = 150
	takeProfit = 0.004 // 千分之四
	stopLoss   = 0.002 // 千分之二
	quantity   = 150   // 交易数量

)

var (
	client            *futures.Client
	mu                sync.Mutex
	rsiValues         []float64
	closes            []float64
	entryPrice        float64
	positionOpen      bool
	orderID           int64
	wsStop            chan struct{}
	doneC             chan struct{}
	averageTop5RSI    float64
	averageBottom5RSI float64
	lostCount         int
	protectTime       time.Time
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

// 订阅K线WebSocket
func subscribeKlineWebSocket() {
	for {
		wsKlineHandler := func(event *futures.WsKlineEvent) {
			closePrice := parseFloat(event.Kline.Close)
			// 只在K线结束时处理
			if event.Kline.IsFinal {
				closes[len(closes)-1] = closePrice
				closes = append(closes, closePrice)
			} else {
				closes[len(closes)-1] = closePrice
			}

			updateRSI()
			checkTradingSignal(closePrice)

		}

		errHandler := func(err error) {
			log.Printf("WebSocket error: %v", err)
		}
		var err error
		doneC, wsStop, err = futures.WsKlineServe(symbol, interval, wsKlineHandler, errHandler)
		if err != nil {
			log.Fatalf("Failed to start WebSocket: %v", err)
		}
		// defer stopC()

		log.Println("WebSocket connected, waiting for kline data...")

		// 保持WebSocket连接
		<-doneC
		log.Println("subscribeKlineWebSocket reconnet")
		time.Sleep(3 * time.Second)
	}
}

func wsUserReConnect() {
	for {
		time.Sleep(55 * time.Minute)
		fmt.Println("1hour reconnect wsUser")
		wsStop <- struct{}{}
	}
}

// 检查交易信号
func checkTradingSignal(currentPrice float64) {
	mu.Lock()
	defer mu.Unlock()
	if positionOpen {
		return
	}
	if lostCount > 2 && time.Now().Before(protectTime.Add(time.Minute*10)) {
		return
	}
	// 找到最高的5个RSI值
	top5RSI := getTop4RSI(rsiValues)
	bottom5RSI := getBottom4RSI(rsiValues)

	// 计算最高5个RSI的平均值
	averageTop5RSI = calculateAverage(top5RSI)
	averageBottom5RSI = calculateAverage(bottom5RSI)

	// 获取当前RSI值
	currentRSI := rsiValues[len(rsiValues)-1]

	CheckShort(currentRSI, averageTop5RSI, currentPrice)
	CheckLong(currentRSI, averageBottom5RSI, currentPrice)

}
func CheckShort(currentRSI, averageTop5RSI, currentPrice float64) {
	// 判断是否做空
	if currentRSI > averageTop5RSI {
		log.Println("当前RSI高于最高5个RSI的平均值，执行做空操作")
		entryPrice = currentPrice
		positionOpen = true

		// 执行做空操作
		order, err := openPosition(futures.PositionSideTypeShort, quantity)
		if err != nil {
			log.Printf("Failed to create order: %v", err)
			return
		}
		orderID = order.OrderID
		orderFilledChan := make(chan []string)
		go CheckOrderById(symbol, order.OrderID, orderFilledChan)
		values := <-orderFilledChan
		if len(values) == 3 {
			entryPrice, _ = strconv.ParseFloat(values[2], 64)
			log.Printf("做空订单已创建，订单ID: %d, 成交价格: %f\n", orderID, entryPrice)
			// amount, _ := strconv.ParseFloat(values[0], 64)
			// quantity, _ := strconv.ParseFloat(values[1], 64)
			// price := functions.RoundStepSize(amount/quantity, configuration.PriceFilterMap[pair])
		}
		// 启动止盈止损监控
		go monitorShortTPSL(entryPrice)
	}
}
func CheckLong(currentRSI, averageBottom5RSI, currentPrice float64) {
	// 判断是否做空
	if currentRSI < averageBottom5RSI {
		log.Println("当前RSI低于最高5个RSI的平均值，执行做多操作")
		entryPrice = currentPrice
		positionOpen = true

		// 执行做空操作
		order, err := openPosition(futures.PositionSideTypeLong, quantity)
		if err != nil {
			log.Printf("Failed to create order: %v", err)
			return
		}
		orderID = order.OrderID

		orderFilledChan := make(chan []string)
		go CheckOrderById(symbol, order.OrderID, orderFilledChan)
		values := <-orderFilledChan
		if len(values) == 3 {
			entryPrice, _ = strconv.ParseFloat(values[2], 64)
			log.Printf("做多订单已创建，订单ID: %d, 成交价格: %f\n", orderID, entryPrice)
			// amount, _ := strconv.ParseFloat(values[0], 64)
			// quantity, _ := strconv.ParseFloat(values[1], 64)
			// price := functions.RoundStepSize(amount/quantity, configuration.PriceFilterMap[pair])
		}
		// 启动止盈止损监控
		go monitorLongTPSL(entryPrice)
	}
}

// 监控止盈止损
func monitorLongTPSL(entryPrice float64) {
	for {
		currentPrice := closes[len(closes)-1]
		// 计算盈亏比例
		profit := (currentPrice - entryPrice) / entryPrice

		if profit >= takeProfit {
			log.Println("达到止盈条件，平仓")
			closePosition(futures.PositionSideTypeLong, currentPrice)
			lostCount = 0
			goto EXIT
		} else if profit <= -stopLoss {
			log.Println("达到止损条件，平仓")
			closePosition(futures.PositionSideTypeLong, currentPrice)
			lostCount++
			if lostCount > 2 {
				protectTime = time.Now()
			}
			goto EXIT
		}

	}
EXIT:
	return
}

// 监控止盈止损
func monitorShortTPSL(entryPrice float64) {
	for {
		currentPrice := closes[len(closes)-1]
		// 计算盈亏比例
		profit := (entryPrice - currentPrice) / entryPrice

		if profit >= takeProfit {
			log.Println("达到止盈条件，平仓")
			closePosition(futures.PositionSideTypeShort, currentPrice)
			lostCount = 0
			goto EXIT
		} else if profit <= -stopLoss {
			log.Println("达到止损条件，平仓")
			closePosition(futures.PositionSideTypeShort, currentPrice)
			lostCount++
			if lostCount > 2 {
				protectTime = time.Now()
			}
			goto EXIT
		}

	}
EXIT:
	return
}

func CheckOrderById(pair string, orderId int64, orderFilledChan chan []string) {
	var order *futures.Order
	var err error
	for {
		order, err = client.NewGetOrderService().Symbol(pair).
			OrderID(orderId).Do(context.Background())
		if err != nil {
			fmt.Println("GetOrderById::error::", err)
		}
		if order.Status == futures.OrderStatusTypeFilled {
			break
		}
		time.Sleep(time.Second * 1)
	}
	orderFilledChan <- []string{order.CumQuote, order.ExecutedQuantity, order.AvgPrice}
}
