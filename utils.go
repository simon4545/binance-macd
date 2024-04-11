package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/markcheno/go-talib"
	"github.com/remeh/sizedwaitgroup"
	"github.com/shopspring/decimal"
)

var Period = "5m"
var SecondsPerUnit map[string]int = map[string]int{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}
var lotSizeMap map[string]float64
var priceFilterMap map[string]float64

func RoundStepSize(quantity float64, step_size float64) float64 {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size))).InexactFloat64()
}

func RoundStepSizeDecimal(quantity float64, step_size float64) decimal.Decimal {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size)))
}

func getBalance(client *binance.Client, token string) float64 {
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

func GetSymbolInfo(client *binance.Client) {
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		print("Error fetching exchange info:", err)
		os.Exit(1)
	}
	for _, s := range info.Symbols {
		if strings.HasSuffix(s.Symbol, "USDT") {
			lotSizeFilter := s.LotSizeFilter()
			quantityTickSize, _ := strconv.ParseFloat(lotSizeFilter.StepSize, 64)
			lotSizeMap[s.Symbol] = quantityTickSize
			priceFilter := s.PriceFilter()
			priceTickSize, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
			priceFilterMap[s.Symbol] = priceTickSize
			// return
		}
	}
}

func checkCross(client *binance.Client, symbol string) {
	// defer time.Sleep(4 * time.Second)
	klines, err := client.NewKlinesService().Symbol(symbol + "USDT").Interval(Period).Limit(200).Do(context.Background())
	if err != nil {
		print(err)
		return
	}
	closingPrices := []float64{}
	for _, kline := range klines {
		close, _ := strconv.ParseFloat(kline.Close, 64)
		closingPrices = append(closingPrices, close)
	}
	lastPrice, _ := strconv.ParseFloat(klines[len(klines)-1].Close, 64)
	Handle(config, symbol, lastPrice, closingPrices)
}

func crossover(a, b []float64) bool {
	return a[len(a)-2] < b[len(b)-2] && a[len(a)-1] >= b[len(b)-1]
}

func crossdown(a, b []float64) bool {
	return a[len(a)-2] >= b[len(b)-2] && a[len(a)-1] < b[len(b)-1]
}

func Handle(c *Config, symbol string, lastPrice float64, closingPrices []float64) {
	// var err error
	pair := fmt.Sprintf("%sUSDT", symbol)
	if len(closingPrices) < 30 {
		return
	}
	if lotSizeMap[pair] == 0 {
		fmt.Println("没有拿到精度")
		return
	}
	_, _, hits := talib.Macd(closingPrices, 12, 26, 9)
	// ema10 := talib.Ema(closingPrices, 10)
	// ema26 := talib.Ema(closingPrices, 26)
	// if crossover(ema10, ema26) {
	if hits[len(hits)-2] <= 0 && hits[len(hits)-1] > 0 {
		//条件 总持仓不能超过10支，一支不能买超过6次 ，最近三根k线不能多次交易，本次进场价要低于上次进场价
		if CheckTotalInvestment() && GetInvestmentCount(symbol) < 6 && GetRecentInvestment(symbol, Period) == 0 && InvestmentAvgPrice(symbol, lastPrice) {
			fmt.Println(symbol, time.Now().Format("2006-01-02 15:04:05"), "出现金叉", lastPrice, "投资数", GetInvestmentCount(symbol), "最近是否有投资", GetRecentInvestment(symbol, Period), "持仓平均价", InvestmentAvgPrice(symbol, lastPrice), "总持仓数", CheckTotalInvestment())
			balance := getBalance(client, "USDT")
			if balance == config.Amount {
				fmt.Println(symbol, "余额不足")
				return
			}
			//插入买单
			ret := createMarketOrder(client, pair, strconv.FormatFloat(config.Amount, 'f', -1, 64), "BUY")
			if ret != nil {
				InsertInvestment(symbol, c.Amount, RoundStepSize((c.Amount/lastPrice), lotSizeMap[pair]))
			}
		}
	}
	// if crossdown(ema10, ema26) {
	if hits[len(hits)-2] > 0 && hits[len(hits)-1] <= 0 {
		if GetInvestmentCount(symbol) == 0 {
			return
		}
		fmt.Println(symbol, "出现死叉", "GetSumInvestment", GetSumInvestment(symbol), "GetInvestmentCount", GetInvestmentCount(symbol))
		balance := GetSumInvestmentQuantity(symbol)
		if balance > lotSizeMap[pair] &&
			((balance*lastPrice) > GetSumInvestment(symbol) ||
				GetInvestmentCount(symbol) >= 6) {
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

func CheckCross(client *binance.Client) {
	for {
		fmt.Println(time.Now(), "开启新的一启")
		swg := sizedwaitgroup.New(4)
		for _, s := range symbols {
			swg.Add()
			go func(s string) {
				defer swg.Done()
				checkCross(client, s)
				time.Sleep(time.Millisecond * 100)
			}(s)
		}
		swg.Wait()
		time.Sleep(time.Second * 60)
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
func createMarketOrder(client *binance.Client, pair string, quantity string, side string) (order *binance.CreateOrderResponse) {
	var sideType binance.SideType
	var err error
	if side == "BUY" {
		sideType = binance.SideTypeBuy
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(RandStr(12)).
			Side(sideType).Type(binance.OrderTypeMarket).QuoteOrderQty(quantity).Do(context.Background(), binance.WithRecvWindow(10000))
	} else {
		sideType = binance.SideTypeSell
		quantityF, _ := decimal.NewFromString(quantity)
		step := decimal.NewFromFloat(lotSizeMap[pair])
		quantity = strconv.FormatFloat(RoundStepSize(quantityF.InexactFloat64(), step.InexactFloat64()), 'f', -1, 64)
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(RandStr(12)).
			Side(sideType).Type(binance.OrderTypeMarket).Quantity(quantity).Do(context.Background(), binance.WithRecvWindow(10000))
	}
	if err != nil {
		print("交易出错", err)
		return nil
	}
	return order
}
