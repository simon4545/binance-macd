package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
	"github.com/remeh/sizedwaitgroup"
	"github.com/shopspring/decimal"
)

var SecondsPerUnit map[string]int = map[string]int{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}
var lotSizeMap map[string]float64
var priceFilterMap map[string]float64
var feeMap map[string]float64

var orderLocker sync.Mutex

func RoundStepSize(quantity float64, step_size float64) float64 {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size))).InexactFloat64()
}

func RoundStepSizeDecimal(quantity float64, step_size float64) decimal.Decimal {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size)))
}

func getBalance(client *futures.Client, token string) float64 {
	res, err := client.NewGetBalanceService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1
	}
	balance := 0.0
	for _, s := range res {
		if s.Asset == token {
			balance, _ = strconv.ParseFloat(s.AvailableBalance, 64)
		}
	}
	return balance
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
			lotSizeMap[s.Symbol] = quantityTickSize
			priceFilter := s.PriceFilter()
			priceTickSize, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
			priceFilterMap[s.Symbol] = priceTickSize
		}
		// return
		// }
	}
	for _, s := range config.Symbols {
		rate, err := client.NewCommissionRateService().Symbol(s + "USDT").Do(context.Background())
		if err != nil {
			print("Error fetching trade fee:", err)
			os.Exit(1)
		}
		fee, _ := strconv.ParseFloat(rate.TakerCommissionRate, 64)
		feeMap[s] = fee
	}
}
func checkAtr(client *futures.Client, symbol string) {
	pair := fmt.Sprintf("%sUSDT", symbol)
	if lotSizeMap[pair] == 0 {
		fmt.Println("交易对", pair, "不可交易")
		return
	}
	klines, err := client.NewKlinesService().Symbol(pair).Interval("1h").Limit(100).Do(context.Background())
	if err != nil {
		print(err)
		return
	}
	closingPrices := []float64{}
	highPrices := []float64{}
	lowPrices := []float64{}
	for _, kline := range klines {
		close, _ := strconv.ParseFloat(kline.Close, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		closingPrices = append(closingPrices, close)
		highPrices = append(highPrices, high)
		lowPrices = append(lowPrices, low)
	}
	atr := talib.Atr(highPrices, lowPrices, closingPrices, 12)
	atrMap[symbol] = atr[len(atr)-1]
	// fmt.Println(symbol, "atr", atrMap[symbol])
}
func CheckAtr(client *futures.Client) {
	for {
		fmt.Println(time.Now(), "开启新的一启")
		for _, s := range symbols {
			checkAtr(client, s)
			time.Sleep(time.Millisecond * 100)
		}
		time.Sleep(time.Second * 20)
	}
}
func checkCross(client *futures.Client, symbol string) {
	// defer time.Sleep(4 * time.Second)
	klines, err := client.NewKlinesService().Symbol(symbol + "USDT").Interval(config.Period).Limit(200).Do(context.Background())
	if err != nil {
		print(err)
		return
	}
	closingPrices := []float64{}
	highPrices := []float64{}
	lowPrices := []float64{}
	for _, kline := range klines {
		close, _ := strconv.ParseFloat(kline.Close, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		closingPrices = append(closingPrices, close)
		highPrices = append(highPrices, high)
		lowPrices = append(lowPrices, low)
	}
	lastPrice, _ := strconv.ParseFloat(klines[len(klines)-1].Close, 64)
	Handle(config, symbol, lastPrice, closingPrices, highPrices, lowPrices)
}

func crossover(a, b []float64) bool {
	return a[len(a)-2] < b[len(b)-2] && a[len(a)-1] >= b[len(b)-1]
}

func crossdown(a, b []float64) bool {
	return a[len(a)-2] >= b[len(b)-2] && a[len(a)-1] < b[len(b)-1]
}

func Handle(c *Config, symbol string, lastPrice float64, closingPrices, highPrices, lowPrices []float64) {
	orderLocker.Lock()
	defer orderLocker.Unlock()
	pair := fmt.Sprintf("%sUSDT", symbol)
	if len(closingPrices) < 30 {
		return
	}
	//TODO 如果FDUSD交易对，fee本身就是0，这里需要做一次单独处理
	if lotSizeMap[pair] == 0 || atrMap[symbol] == 0 || feeMap[symbol] == 0 {
		fmt.Println("没有拿到精度")
		return
	}
	// _, _, hits := talib.Macd(closingPrices, 12, 26, 9)
	ema6 := talib.Ema(closingPrices, 6)
	ema26 := talib.Ema(closingPrices, 26)

	investCount := GetInvestmentCount(symbol)
	sumInvestment := GetSumInvestment(symbol)
	balance := GetSumInvestmentQuantity(symbol)
	rate := float64(investCount/2.0)/100.0 + 1
	level := config.Level
	if crossover(ema6, ema26) {
		// if hits[len(hits)-2] <= 0 && hits[len(hits)-1] > 0 {
		hasRecentInvestment := GetRecentInvestment(symbol, config.Period)
		lowThanInvestmentAvgPrice := InvestmentAvgPrice(symbol, lastPrice)
		checkTotalInvestment := CheckTotalInvestment()
		//条件 总持仓不能超过10支，一支不能买超过6次 ，最近5根k线不能多次交易，本次进场价要低于上次进场价
		fmt.Println(symbol, time.Now().Format("2006-01-02 15:04:05"), "出现金叉", lastPrice, "投资数", investCount, "最近是否有投资", hasRecentInvestment, "持仓平均价", lowThanInvestmentAvgPrice, "总持仓数", checkTotalInvestment)
		if investCount < level && hasRecentInvestment == 0 && lowThanInvestmentAvgPrice {
			if investCount <= 0 && !checkTotalInvestment {
				fmt.Println(symbol, "投资达到总数")
				return
			}
			balance := getBalance(client, "USDT")
			if balance == config.Amount {
				fmt.Println(symbol, "余额不足")
				return
			}
			//插入买单
			quantity := RoundStepSize(config.Amount/lastPrice, lotSizeMap[pair])
			orderFilledChan := make(chan []string)
			order := createMarketOrder(client, pair, strconv.FormatFloat(quantity, 'f', -1, 64), "BUY")
			if order != nil {
				go CheckOrderById(pair, order.OrderID, orderFilledChan)
				values := <-orderFilledChan
				if len(values) == 3 {
					fmt.Println(symbol, values)
					//TODO 市价单查不出交易的数量只能返回平均价和总投入
					amount, _ := strconv.ParseFloat(values[0], 64)
					price, _ := strconv.ParseFloat(values[2], 64)
					quantity := amount / price
					// quantity = quantity * (1 - feeMap[pair])
					InsertInvestment(symbol, amount, RoundStepSize(quantity, lotSizeMap[pair]), price)
				}
			}
		}
	}
	if investCount > 0 && balance > lotSizeMap[pair] {
		atrRate := atrMap[symbol] / lastPrice
		if (balance*lastPrice) >= sumInvestment*(1+atrRate) || (crossdown(ema6, ema26) && ((balance*lastPrice) > sumInvestment*rate || investCount >= level)) {
			// if hits[len(hits)-2] > 0 && hits[len(hits)-1] <= 0 {
			// fmt.Print("出现死叉", lotSizeMap[pair])

			fmt.Println(symbol, "出现死叉", "GetSumInvestment", sumInvestment, "GetInvestmentCount", investCount)
			quantity := RoundStepSize(balance, lotSizeMap[pair])
			fmt.Println(symbol, "quantity", quantity)
			// 插入卖单
			ret := createMarketOrder(client, pair, strconv.FormatFloat(quantity, 'f', -1, 64), "SELL")
			if ret != nil {
				ClearHistory(symbol)
			}
		}
	}
}

func CheckCross(client *futures.Client) {
	for {
		fmt.Println(time.Now(), "开启新的一启")
		swg := sizedwaitgroup.New(4)
		for _, s := range symbols {
			swg.Add()
			go func(s string) {
				defer swg.Done()
				pair := fmt.Sprintf("%sUSDT", s)
				if lotSizeMap[pair] != 0 {
					checkCross(client, s)
					time.Sleep(time.Millisecond * 100)
				} else {
					fmt.Println("交易对", pair, "不可交易")
					return
				}
			}(s)
		}
		swg.Wait()
		time.Sleep(time.Second * 10)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStr(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	str := fmt.Sprintf("SIM-%s", string(b))
	return str
}
func createMarketOrder(client *futures.Client, pair string, quantity string, side string) (order *futures.CreateOrderResponse) {
	var sideType futures.SideType
	var err error
	if side == "BUY" {
		sideType = futures.SideTypeBuy
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(RandStr(12)).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	} else {
		sideType = futures.SideTypeSell
		quantityF, _ := decimal.NewFromString(quantity)
		step := decimal.NewFromFloat(lotSizeMap[pair])
		quantity = strconv.FormatFloat(RoundStepSize(quantityF.InexactFloat64(), step.InexactFloat64()), 'f', -1, 64)
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(RandStr(12)).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	}
	if err != nil {
		fmt.Println("交易出错", err)
		return nil
	}
	return order
}

func CheckOrderById(pair string, orderId int64, orderFilledChan chan []string) {
	var order *futures.Order
	var err error
	for {
		order, err = client.NewGetOrderService().Symbol(pair).OrderID(orderId).Do(context.Background())
		if err != nil {
			fmt.Println("GetOrderById::error::", err)
		}
		if order.Status == futures.OrderStatusTypeFilled {
			break
		}
		time.Sleep(time.Second * 1)
	}
	fmt.Println(order)
	orderFilledChan <- []string{order.CumQuote, order.CumQuantity, order.AvgPrice}
}
