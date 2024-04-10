package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/markcheno/go-talib"
	"github.com/remeh/sizedwaitgroup"
	"github.com/shopspring/decimal"
)

var Period = "4h"
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
	var err error
	tokenpairs := strings.Split(symbol, "_")
	ema10 := talib.Ema(closingPrices, 10)
	ema26 := talib.Ema(closingPrices, 26)
	if crossover(ema10, ema26) {
		fmt.Println(symbol, time.Now().Format("2006-01-02 15:04:05"), "出现金叉", lastPrice, "投资数", GetInvestmentCount(tokenpairs[0]), "最近是否有投资", GetRecentInvestment(tokenpairs[0], Period), "持仓平均价", InvestmentAvgPrice(tokenpairs[0], lastPrice))

		if GetInvestmentCount(tokenpairs[0]) < 6 && GetRecentInvestment(tokenpairs[0], Period) == 0 && InvestmentAvgPrice(tokenpairs[0], lastPrice) {
			balance := getBalance(client, tokenpairs[1])
			if balance == -1 {
				fmt.Println(symbol, err)
				os.Exit(1)
				return
			}
			//插入买单
			InsertInvestment(tokenpairs[0], c.Amount, RoundStepSize((c.Amount/lastPrice), lotSizeMap[symbol]))
		}
	}
	if crossdown(ema10, ema26) {
		fmt.Println(symbol, "出现死叉")
		if GetInvestmentCount(tokenpairs[0]) == 0 {
			return
		}
		balance := GetSumInvestmentQuantity(tokenpairs[0])
		if balance > lotSizeMap[symbol] && ((balance*lastPrice) > GetSumInvestment(tokenpairs[0]) ||
			GetInvestmentCount(tokenpairs[0]) >= 6) {
			quantity := RoundStepSize(balance, lotSizeMap[symbol])
			fmt.Println(symbol, "quantity", quantity)
			// 插入卖单
		}
	}
}

func CheckCross(client *binance.Client) {
	for {
		swg := sizedwaitgroup.New(1)
		for _, s := range config.Symbols {
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
