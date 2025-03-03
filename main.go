package main

import (
	"context"
	"fmt"
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
	// 创建一个定时器，每5秒触发一次
	// ticker := time.NewTicker(interval)
	// defer ticker.Stop()

	// // 无限循环，每5秒执行一次检测
	// for range ticker.C {
	// 	fmt.Println("Starting detection...")

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
