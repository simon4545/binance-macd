package bn

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
	"github.com/simon4545/binance-macd/functions"
	"github.com/spf13/cast"
)

func GetBalance(client *binance.Client, token string) float64 {
	res, err := client.NewGetAccountService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1
	}
	balance := 0.0
	// fmt.Println(res.Balances)
	for _, s := range res.Balances {
		if s.Asset == token {
			balance, _ = strconv.ParseFloat(s.Free, 64)
		}
	}
	return balance
}

// 执行交易
func placeOrder(client *futures.Client, symbol string, side futures.SideType, position futures.PositionSideType) error {
	// 计算合约数量
	quantity := SymbolDebet[symbol]

	// 下单
	order, err := client.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		PositionSide(position).
		Type(futures.OrderTypeMarket).
		Quantity(quantity).
		Do(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Order placed: %v\n", order)
	orderID := order.OrderID
	orderFilledChan := make(chan []string)
	go CheckOrderById(symbol, order.OrderID, orderFilledChan)
	values := <-orderFilledChan
	if len(values) == 3 {
		entryPrice, _ := strconv.ParseFloat(values[2], 64)
		log.Printf("做空订单已创建，订单ID: %d, 成交价格: %f\n", orderID, entryPrice)
		setTakeProfitAndStopLoss(client, symbol, position, entryPrice, quantity)
		// amount, _ := strconv.ParseFloat(values[0], 64)
		// quantity, _ := strconv.ParseFloat(values[1], 64)
		// price := functions.RoundStepSize(amount/quantity, configuration.PriceFilterMap[pair])
	}
	return nil
}

// 设置止盈和止损
func setTakeProfitAndStopLoss(client *futures.Client, symbol string, position futures.PositionSideType, entryPrice float64, quantity string) error {
	var side futures.SideType
	// 计算止盈和止损价格
	var takeProfitPrice, stopLossPrice float64
	if position == futures.PositionSideTypeLong {
		// 做多：止盈为当天振幅的1/4，止损为最近30天的最低价或10%
		takeProfitPrice = entryPrice + Atrs[symbol]*2
		stopLossPrice = entryPrice - Atrs[symbol]
		side = futures.SideTypeSell
	} else {
		// 做空：止盈为当天振幅的1/4，止损为最近30天的最高价或10%
		takeProfitPrice = entryPrice - Atrs[symbol]*2
		stopLossPrice = entryPrice + Atrs[symbol]
		side = futures.SideTypeBuy
	}
	takeProfitPrice = functions.RoundStepSize(takeProfitPrice, SymbolStepSize[symbol])
	stopLossPrice = functions.RoundStepSize(stopLossPrice, SymbolStepSize[symbol])
	// 设置止盈单
	_, err := client.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		PositionSide(position).
		Type(futures.OrderTypeTakeProfitMarket).
		StopPrice(cast.ToString(takeProfitPrice)).
		Quantity(quantity).
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
		StopPrice(cast.ToString(stopLossPrice)).
		Quantity(quantity).
		Do(context.Background())
	if err != nil {
		return fmt.Errorf("error setting stop loss order: %v", err)
	}

	fmt.Printf("Take profit and stop loss set. Take profit: %.2f, Stop loss: %.2f\n", takeProfitPrice, stopLossPrice)
	return nil
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
		time.Sleep(time.Millisecond * 500)
	}
	orderFilledChan <- []string{order.CumQuote, order.ExecutedQuantity, order.AvgPrice}
}
func CheckATR() {
	for _, k := range Symbols {
		Atrs[k] = calculateAtr(k, 30)
	}
	fmt.Println("Atr", Atrs)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for t := range ticker.C {
		for _, k := range Symbols {
			Atrs[k] = calculateAtr(k, 30)
		}
		fmt.Printf("执行任务，当前时间: %v\n", t)
	}
}
func getKlines(symbol, interval string, limit int) ([]*futures.Kline, error) {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return klines, nil
}

// 计算ATR
func calculateAtr(symbol string, limit int) float64 {
	klines, err := getKlines(symbol, "4h", limit)
	if err != nil {
		return 0
	}
	var closes []float64
	var highs []float64
	var lows []float64
	// var opens []float64
	for _, kline := range klines {
		if kline.CloseTime > time.Now().UnixMilli() {
			continue
		}
		// open, _ := strconv.ParseFloat(kline.Open, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		close, _ := strconv.ParseFloat(kline.Close, 64)
		closes = append(closes, close)
		highs = append(highs, high)
		lows = append(lows, low)
		// opens = append(opens, open)
	}
	atrs := talib.Atr(highs, lows, closes, 14)
	return atrs[len(atrs)-1]
}

// 计算真实波动范围（TR）
func calculateTR(high, low, close float64) float64 {
	tr1 := high - low
	tr2 := math.Abs(high - close)
	tr3 := math.Abs(low - close)
	return math.Max(tr1, math.Max(tr2, tr3))
}

// 计算ATR
func calculateATR(high, low, close []float64, period int) []float64 {
	atr := make([]float64, len(close))
	trSum := 0.0

	for i := 0; i < len(close); i++ {
		if i == 0 {
			// 第一天的TR就是当天的最高价与最低价的差值
			atr[i] = high[i] - low[i]
			trSum += atr[i]
		} else {
			tr := calculateTR(high[i], low[i], close[i-1])
			trSum += tr
			if i < period {
				// 前period天的ATR是TR的简单平均
				atr[i] = trSum / float64(i+1)
			} else {
				// 之后的ATR是前一天的ATR乘以(period-1)加上当天的TR，再除以period
				atr[i] = (atr[i-1]*float64(period-1) + tr) / float64(period)
			}
		}
	}
	return atr
}

func GetSymbolInfo(client *futures.Client) {
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		print("Error fetching exchange info:", err)
		os.Exit(1)
	}
	for _, s := range info.Symbols {
		// if strings.HasSuffix(s.Symbol, "USDT") {
		if s.Status == string(futures.SymbolStatusTypeTrading) {
			lotSizeFilter := s.LotSizeFilter()
			quantityTickSize, _ := strconv.ParseFloat(lotSizeFilter.StepSize, 64)
			LotSizeMap[s.Symbol] = quantityTickSize
			priceFilter := s.PriceFilter()
			priceTickSize, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
			PriceFilterMap[s.Symbol] = priceTickSize
		}
		// return
		// }
	}
}
func GetFeeInfo(client *futures.Client, symbols []string) {
	for _, s := range symbols {
		if FeeMap[s] != 0 {
			continue
		}
		rate, err := client.NewCommissionRateService().Symbol(s).Do(context.Background())
		if err != nil {
			print("Error fetching trade fee:", err)
			os.Exit(1)
		}
		fee, _ := strconv.ParseFloat(rate.TakerCommissionRate, 64)
		FeeMap[s] = fee
	}
}
