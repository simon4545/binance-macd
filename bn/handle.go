package bn

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/markcheno/go-talib"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/configuration"
	"github.com/simon4545/binance-macd/db"
	"github.com/simon4545/binance-macd/functions"
)

var orderLocker sync.Mutex
var client *binance.Client
var c *configuration.Config

func Init(config *configuration.Config) {
	c = config
	db.InitDB()
	client = binance.NewClient(config.BAPI_KEY, config.BAPI_SCRET)
	GetSymbolInfo(client)
	InitWS()
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for t := range ticker.C {
			fmt.Println("定时任务执行，当前时间：", t)
			for _, sym := range c.Symbols {
				assetInfo := AssetInfo[sym]
				Handle(sym, assetInfo.Price, assetInfo.Close, assetInfo.High, assetInfo.Low)
			}
		}
	}()
	// go functions.CheckCross(client, config.Symbols, config, Handle)
}
func Handle(pair string, lastPrice float64, closingPrices, highPrices, lowPrices []float64) {
	orderLocker.Lock()
	defer orderLocker.Unlock()
	if len(closingPrices) < 30 {
		return
	}
	//TODO 如果FDUSD交易对，fee本身就是0，这里需要做一次单独处理
	if configuration.LotSizeMap[pair] == 0 || configuration.FeeMap[pair] == 0 {
		fmt.Println("没有拿到精度")
		return
	}
	symbol, _ := functions.SplitSymbol(pair)

	fastSignal, slowSignal, _ := talib.Macd(closingPrices, 12, 26, 9)

	invests := db.GetInvestments(symbol)

	investCount := getInvestmentCount(invests)
	sumInvestment := getSumInvestment(invests)
	balance := getSumInvestmentQuantity(invests)
	invest := recentInvestment(invests)

	level := c.Level
	takeprofit := balance*lastPrice - sumInvestment
	fmt.Println("浮动盈亏", pair, functions.RoundStepSize(takeprofit, 0.1))
	// 条件
	// 总持仓不能超过10支，
	// 一支不能买超过6次
	// 最近5根k线不能多次交易
	// 本次进场价要低于上次进场价
	if len(invests) == 0 || (len(invests) > 0 && lastPrice < invest.UnitPrice*(1-c.ForceSell)) {
		fmt.Println(pair, time.Now().Format("2006-01-02 15:04:05"), "进入强制买入条件")
		createOrder(c, investCount, pair)
	}

	if functions.Crossover(fastSignal, slowSignal) {
		recentInvestment := recentInvestmentPrice(invests, c.Period)

		fmt.Println(pair, time.Now().Format("2006-01-02 15:04:05"), "出现金叉", lastPrice,
			"投资数", investCount,
			"最近持仓平均价", recentInvestment)

		if investCount < level && recentInvestment == -1 {
			if invest != nil && lastPrice > invest.UnitPrice*(1-c.PriceProtect) {
				fmt.Println(pair, "价格过于接近，不建仓")
				return
			}
			createOrder(c, investCount, pair)
		}
	}

	if investCount > 0 && balance > configuration.LotSizeMap[pair] {
		// 如果当前价格高于所有仓位的平均价的1.15倍
		if investCount > 2 && (balance*lastPrice) >= sumInvestment*(1+c.ForceSell) {
			fmt.Println(pair, "出现死叉", balance, lastPrice, "GetSumInvestment", sumInvestment, "GetInvestmentCount", investCount)
			quantity := functions.RoundStepSize(balance, configuration.LotSizeMap[pair])
			fmt.Println(pair, "quantity", quantity)
			// earn := quantity*lastPrice - sumInvestment
			// // 插入卖单
			// ret := createMarketOrder(client, pair, strconv.FormatFloat(quantity, 'f', -1, 64), "SELL")
			// if ret != nil {
			// 	ClearHistory(symbol, earn)
			// }
			amount := createSOrder(strconv.FormatFloat(quantity, 'f', -1, 64), pair)
			db.ClearHistory(symbol, amount-sumInvestment-amount*configuration.FeeMap[pair])
			return
		}
		//如果出现死叉
		//如果现价比建仓价高10%
		for _, v := range invests {
			quantity := functions.RoundStepSize(v.Quantity, configuration.LotSizeMap[pair])
			//如果出现死叉
			//如果现价比建仓价高20%
			if (functions.Crossdown(fastSignal, slowSignal) && quantity*lastPrice > v.Amount*1.01) ||
				(quantity*lastPrice > v.Amount*(1+c.ForceSell)) {
				fmt.Println(pair, "出现死叉", balance, lastPrice, "GetSumInvestment", sumInvestment, "GetInvestmentCount", investCount)
				// 插入卖单
				amount := createSOrder(strconv.FormatFloat(quantity, 'f', -1, 64), pair)
				// ClearHistory(symbol, amount-sumInvestment)
				db.ClearHistoryById(v.ID, symbol, amount-v.Amount-amount*configuration.FeeMap[pair])
				// earn := quantity*lastPrice - v.Amount
				// ret := createMarketOrder(client, pair, strconv.FormatFloat(quantity, 'f', -1, 64), "SELL")
				// if ret != nil {
				// 	ClearHistoryById(v.ID, symbol, earn)
				// }
			}
		}
	}
}
func createSOrder(quantity string, pair string) (amount float64) {
	orderFilledChan := make(chan []string)
	order := createMarketOrder(client, pair, quantity, "SELL")
	if order != nil {
		go CheckOrderById(pair, order.OrderID, orderFilledChan)
		values := <-orderFilledChan
		if len(values) == 2 {
			amount, _ = strconv.ParseFloat(values[0], 64)
			// quantity, _ := strconv.ParseFloat(values[1], 64)
			// quantity = quantity * (1 - configuration.FeeMap[pair])
			// InsertInvestment(symbol, amount, RoundStepSize(quantity, configuration.LotSizeMap[pair]), price)
		}
	}
	return
}

func createOrder(c *configuration.Config, investCount int, pair string) {
	_, quote := functions.SplitSymbol(pair)
	balance := GetBalance(client, quote)
	if balance < c.Amount {
		fmt.Println(pair, "余额不足")
		return
	}
	//插入买单
	orderFilledChan := make(chan []string)
	needToBuy := c.Amount
	if investCount > 0 {
		needToBuy = functions.RoundStepSize(needToBuy*(1+c.Multi*float64(investCount)), configuration.LotSizeMap[pair])
	}
	order := createMarketOrder(client, pair, strconv.FormatFloat(needToBuy, 'f', -1, 64), "BUY")
	if order != nil {
		go CheckOrderById(pair, order.OrderID, orderFilledChan)
		values := <-orderFilledChan
		if len(values) == 2 {
			fmt.Println(pair, values)
			amount, _ := strconv.ParseFloat(values[0], 64)
			quantity, _ := strconv.ParseFloat(values[1], 64)
			price := functions.RoundStepSize(amount/quantity, configuration.PriceFilterMap[pair])
			quantity = quantity * (1 - configuration.FeeMap[pair])
			s, _ := functions.SplitSymbol(pair)
			db.InsertInvestment(s, amount, functions.RoundStepSize(quantity, configuration.LotSizeMap[pair]), price)
		}
	}
}

func createMarketOrder(client *binance.Client, pair string, quantity string, side string) (order *binance.CreateOrderResponse) {
	var sideType binance.SideType
	var err error
	if side == "BUY" {
		sideType = binance.SideTypeBuy
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(functions.RandStr(12)).
			Side(sideType).Type(binance.OrderTypeMarket).QuoteOrderQty(quantity).Do(context.Background(), binance.WithRecvWindow(10000))
	} else {
		sideType = binance.SideTypeSell
		quantityF, _ := decimal.NewFromString(quantity)
		step := decimal.NewFromFloat(configuration.LotSizeMap[pair])
		quantity = strconv.FormatFloat(functions.RoundStepSize(quantityF.InexactFloat64(), step.InexactFloat64()), 'f', -1, 64)
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(functions.RandStr(12)).
			Side(sideType).Type(binance.OrderTypeMarket).Quantity(quantity).Do(context.Background(), binance.WithRecvWindow(10000))
	}
	if err != nil {
		print("交易出错", err)
		return nil
	}
	return order
}

func CheckOrderById(pair string, orderId int64, orderFilledChan chan []string) {
	var order *binance.Order
	var err error
	for {
		order, err = client.NewGetOrderService().Symbol(pair).
			OrderID(orderId).Do(context.Background())
		if err != nil {
			fmt.Println("GetOrderById::error::", err)
		}
		if order.Status == binance.OrderStatusTypeFilled {
			break
		}
		time.Sleep(time.Second * 1)
	}
	orderFilledChan <- []string{order.CummulativeQuoteQuantity, order.ExecutedQuantity}
}
