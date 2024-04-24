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

func crossover(a, b []float64) bool {
	return a[len(a)-2] < b[len(b)-2] && a[len(a)-1] >= b[len(b)-1]
}

func crossdown(a, b []float64) bool {
	return a[len(a)-2] >= b[len(b)-2] && a[len(a)-1] < b[len(b)-1]
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
