package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/simon4545/binance-macd/bn"
	"github.com/spf13/cast"
)

const (
	apiKey     = "xzJKM9OUwYXxVrOpG9474d2Tgqx57QyABMzIekxXXzDSRNN5ClsNYlDblVVDqaNx"
	secretKey  = "NG7W8uzFSu3PGnIx3lAyxIU232rhrQGsIz8n124A5eIlGeKHRnxKNji3V1cLgyzf"
	symbol     = "BTCUSDT"
	usdtAmount = 1000.0
	interval   = 5 * time.Second // 检测间隔时间

)

var (
	Symbols      = []string{"BTCUSDT", "XRPUSDT", "SOLUSDT", "DOGEUSDT"}
	client       *futures.Client
	mu           sync.Mutex
	orderID      int64
	wsStop       chan struct{}
	doneC        chan struct{}
	currentPrice float64
	entryPrice   float64
	protectTime  time.Time
	klineData    []*futures.Kline
)

func main() {
	// 初始化币安客户端
	client = futures.NewClient(apiKey, secretKey)
	bn.Init(client)
	// // 创建一个定时器，每5秒触发一次
	// ticker := time.NewTicker(interval)
	// defer ticker.Stop()

	// // 无限循环，每5秒执行一次检测
	// for range ticker.C {
	// 	fmt.Println("Starting detection...")

	// 	// 1. 检测当前是否有持仓
	// 	hasPosition, err := checkPosition(client)
	// 	if err != nil {
	// 		log.Printf("Error checking position: %v\n", err)
	// 		continue
	// 	}
	// 	if hasPosition {
	// 		fmt.Println("Already have a position, skipping...")
	// 		continue
	// 	}

	// 	// 2. 获取最近30天的K线数据
	// 	klineData, err = getKlineData(client, 30)
	// 	if err != nil {
	// 		log.Printf("Error getting kline data: %v\n", err)
	// 		continue
	// 	}

	// 	// 计算前四最大跌幅和涨幅
	// 	top4Drops, top4Rises := calculateTop4DropsAndRises(klineData)

	// 	// 计算平均跌幅和涨幅
	// 	avgDrop := calculateAverage(top4Drops)
	// 	avgRise := calculateAverage(top4Rises)

	// 	// 获取今天的涨跌幅
	// 	todayChange := getTodayChange(klineData)

	// 	// 判断是否达到条件并执行交易
	// 	if todayChange <= avgDrop {
	// 		fmt.Println("Today's drop meets the condition, going long...")
	// 		err = placeOrder(client, futures.SideTypeBuy, futures.PositionSideTypeLong)
	// 		if err != nil {
	// 			log.Printf("Error placing long order: %v\n", err)
	// 		}
	// 	} else if todayChange >= avgRise {
	// 		fmt.Println("Today's rise meets the condition, going short...")
	// 		err = placeOrder(client, futures.SideTypeSell, futures.PositionSideTypeShort)
	// 		if err != nil {
	// 			log.Printf("Error placing short order: %v\n", err)
	// 		}
	// 	} else {
	// 		fmt.Println("No trading condition met today.")
	// 	}

	// 	fmt.Println("Detection completed. Waiting for next interval...")
	// }

	// 启动WebSocket订阅K线数据
	// go subscribeKlineWebSocket()
	// go wsUserReConnect()
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
				// highPrice := parseFloat(event.Kline.High)
				// lowPrice := parseFloat(event.Kline.Low)

				// // closes[len(closes)-1] = closePrice
				// // highs[len(highs)-1] = highPrice
				// // lows[len(lows)-1] = lowPrice
				// closes = append(closes, closePrice)
				// highs = append(highs, highPrice)
				// lows = append(lows, lowPrice)
			}
			currentPrice = closePrice
		}

		errHandler := func(err error) {
			log.Printf("WebSocket error: %v", err)
		}
		var err error
		doneC, wsStop, err = futures.WsKlineServe(symbol, "1m", wsKlineHandler, errHandler)
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

// 检测当前是否有持仓
func checkPosition(client *futures.Client) (bool, error) {
	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return false, err
	}

	for _, position := range positions {
		PositionAmt := cast.ToFloat64(position.PositionAmt)
		if position.Symbol == symbol && PositionAmt != 0 {
			return true, nil
		}
	}
	return false, nil
}

// 获取最近30天的K线数据
func getKlineData(client *futures.Client, days int) ([]*futures.Kline, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)
	klines, err := client.NewKlinesService().
		Symbol(symbol).
		Interval("1d").
		StartTime(startTime.Unix() * 1000).
		EndTime(endTime.Unix() * 1000).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
	return klines, nil
}

// 计算前四最大跌幅和涨幅
func calculateTop4DropsAndRises(klines []*futures.Kline) ([]float64, []float64) {
	var drops, rises []float64

	for _, kline := range klines {
		open, _ := strconv.ParseFloat(kline.Open, 64)
		close, _ := strconv.ParseFloat(kline.Close, 64)
		change := (close - open) / open * 100

		if change < 0 {
			drops = append(drops, change)
		} else {
			rises = append(rises, change)
		}
	}

	sort.Float64s(drops)
	sort.Float64s(rises)
	sort.Slice(rises, func(i, j int) bool {
		return rises[i] > rises[j] // 倒序排序
	})
	if len(drops) > 4 {
		drops = drops[:4]
	}
	if len(rises) > 4 {
		rises = rises[:4]
	}

	return drops, rises
}

// 获取今天的涨跌幅
func getTodayChange(klines []*futures.Kline) float64 {
	lastKline := klines[len(klines)-1]
	open, _ := strconv.ParseFloat(lastKline.Open, 64)
	close, _ := strconv.ParseFloat(lastKline.Close, 64)
	return (close - open) / open * 100
}

// 执行交易
func placeOrder(client *futures.Client, side futures.SideType, position futures.PositionSideType) error {
	// 获取当前价格
	ticker, err := client.NewListPricesService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return err
	}
	price, _ := strconv.ParseFloat(ticker[0].Price, 64)

	// 计算合约数量
	quantity := usdtAmount / price

	// 下单
	order, err := client.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		PositionSide(position).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%.3f", quantity)).
		Do(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Order placed: %v\n", order)
	orderID = order.OrderID
	orderFilledChan := make(chan []string)
	go CheckOrderById(symbol, order.OrderID, orderFilledChan)
	values := <-orderFilledChan
	if len(values) == 3 {
		entryPrice, _ = strconv.ParseFloat(values[2], 64)
		log.Printf("做空订单已创建，订单ID: %d, 成交价格: %f\n", orderID, entryPrice)
		setTakeProfitAndStopLoss(client, position, entryPrice, quantity)
		// amount, _ := strconv.ParseFloat(values[0], 64)
		// quantity, _ := strconv.ParseFloat(values[1], 64)
		// price := functions.RoundStepSize(amount/quantity, configuration.PriceFilterMap[pair])
	}
	return nil
}

// 设置止盈和止损
func setTakeProfitAndStopLoss(client *futures.Client, position futures.PositionSideType, entryPrice, quantity float64) error {
	// 计算当天振幅
	lastKline := klineData[len(klineData)-1]
	high, _ := strconv.ParseFloat(lastKline.High, 64)
	low, _ := strconv.ParseFloat(lastKline.Low, 64)
	amplitude := high - low
	var side futures.SideType
	// 计算止盈和止损价格
	var takeProfitPrice, stopLossPrice float64
	if position == futures.PositionSideTypeLong {
		// 做多：止盈为当天振幅的1/4，止损为最近30天的最低价或10%
		takeProfitPrice = entryPrice + amplitude/4
		stopLossPrice = math.Min(getLowestPrice(klineData), entryPrice*0.97)
		side = futures.SideTypeSell
	} else {
		// 做空：止盈为当天振幅的1/4，止损为最近30天的最高价或10%
		takeProfitPrice = entryPrice - amplitude/4
		stopLossPrice = math.Max(getHighestPrice(klineData), entryPrice*1.03)
		side = futures.SideTypeBuy
	}

	// 设置止盈单
	_, err := client.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		PositionSide(position).
		Type(futures.OrderTypeTakeProfitMarket).
		StopPrice(fmt.Sprintf("%.1f", takeProfitPrice)).
		Quantity(fmt.Sprintf("%.3f", quantity)).
		Do(context.Background())
	if err != nil {
		return fmt.Errorf("error setting take profit order: %v", err)
	}

	// 设置止损单
	_, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		PositionSide(position).
		Type(futures.OrderTypeStopMarket).
		StopPrice(fmt.Sprintf("%.1f", stopLossPrice)).
		Quantity(fmt.Sprintf("%.3f", quantity)).
		Do(context.Background())
	if err != nil {
		return fmt.Errorf("error setting stop loss order: %v", err)
	}

	fmt.Printf("Take profit and stop loss set. Take profit: %.2f, Stop loss: %.2f\n", takeProfitPrice, stopLossPrice)
	return nil
}

// 获取最近30天的最低价
func getLowestPrice(klines []*futures.Kline) float64 {
	return cast.ToFloat64(klines[len(klines)-2].Low)
}

// 获取最近30天的最高价
func getHighestPrice(klines []*futures.Kline) float64 {
	return cast.ToFloat64(klines[len(klines)-2].Low)
}

// order, err := openPosition(futures.PositionSideTypeShort, quantity)
// if err != nil {
// 	log.Printf("Failed to create order: %v", err)
// 	return
// }
// orderID = order.OrderID
// orderFilledChan := make(chan []string)
// go CheckOrderById(symbol, order.OrderID, orderFilledChan)
// values := <-orderFilledChan
// if len(values) == 3 {
// 	entryPrice, _ = strconv.ParseFloat(values[2], 64)
// 	log.Printf("做空订单已创建，订单ID: %d, 成交价格: %f\n", orderID, entryPrice)
// 	// amount, _ := strconv.ParseFloat(values[0], 64)
// 	// quantity, _ := strconv.ParseFloat(values[1], 64)
// 	// price := functions.RoundStepSize(amount/quantity, configuration.PriceFilterMap[pair])
// }

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
