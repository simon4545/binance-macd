package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
	"github.com/shopspring/decimal"
)

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
