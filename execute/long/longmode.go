package long

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/config"
	"github.com/simon4545/binance-macd/db"
	"github.com/simon4545/binance-macd/interfacer"
	"github.com/simon4545/binance-macd/utils"
)

func init() {
	t := &LongMode{}
	interfacer.Register("LONG", t)
}

type LongMode struct {
}

func (m *LongMode) Handle(client *futures.Client, c *config.Config, symbol string, lastPrice float64, closingPrices, highPrices, lowPrices []float64) {
	config.OrderLocker.Lock()
	defer config.OrderLocker.Unlock()
	if len(closingPrices) < 30 {
		return
	}
	//TODO 如果FDUSD交易对，fee本身就是0，这里需要做一次单独处理
	if config.LotSizeMap[symbol] == 0 || config.AtrMap[symbol] == 0 || config.FeeMap[symbol] == 0 {
		fmt.Println("没有拿到精度")
		return
	}
	// _, _, hits := talib.Macd(closingPrices, 12, 26, 9)
	ema6 := talib.Ema(closingPrices, 6)
	ema26 := talib.Ema(closingPrices, 26)

	investCount := db.GetInvestmentCount(symbol)
	sumInvestment := db.GetSumInvestment(symbol)
	balance := db.GetSumInvestmentQuantity(symbol)
	rate := float64(investCount/2.0)/100.0 + 1
	level := c.Level
	if utils.Crossover(ema6, ema26) {
		// if hits[len(hits)-2] <= 0 && hits[len(hits)-1] > 0 {
		hasRecentInvestment := db.GetRecentInvestment(symbol, c.Period)
		lowThanInvestmentAvgPrice := db.InvestmentAvgPrice(symbol, lastPrice, 1.02)
		checkTotalInvestment := db.CheckTotalInvestment(c)
		//条件 总持仓不能超过10支，一支不能买超过6次 ，最近5根k线不能多次交易，本次进场价要低于上次进场价
		fmt.Println(symbol, time.Now().Format("2006-01-02 15:04:05"), "出现金叉", lastPrice, "投资数", investCount, "最近是否有投资", hasRecentInvestment, "持仓平均价", lowThanInvestmentAvgPrice, "总持仓数", checkTotalInvestment)
		if investCount < level && hasRecentInvestment == 0 && lowThanInvestmentAvgPrice {
			if investCount <= 0 && !checkTotalInvestment {
				fmt.Println(symbol, "投资达到总数")
				return
			}
			balance := utils.GetBalance(client, "USDT")
			if balance < c.Symbols[symbol].Amount {
				fmt.Println(symbol, "余额不足", balance)
				return
			}
			//插入买单
			m.CreateBuySide(client, c, symbol, c.Symbols[symbol].Amount, lastPrice)
		}
	}
	if investCount > 0 && balance > config.LotSizeMap[symbol] {
		atrRate := config.AtrMap[symbol] / lastPrice
		if (balance*lastPrice) >= sumInvestment*(1+atrRate) || (utils.Crossdown(ema6, ema26) && ((balance*lastPrice) > sumInvestment*rate || investCount >= level)) {
			// if hits[len(hits)-2] > 0 && hits[len(hits)-1] <= 0 {
			// fmt.Print("出现死叉", lotSizeMap[symbol])

			fmt.Println(symbol, "出现死叉", "GetSumInvestment", sumInvestment, "GetInvestmentCount", investCount)
			m.CreateSellSide(client, c, symbol, balance)
		}
	}
}
func (m *LongMode) CreateSellSide(client *futures.Client, c *config.Config, symbol string, balance float64) {
	quantity := utils.RoundStepSize(balance, config.LotSizeMap[symbol])
	fmt.Println(symbol, "quantity", quantity)
	// 插入卖单
	ret := m.createMarketOrder(client, symbol, strconv.FormatFloat(quantity, 'f', -1, 64), "CLOSE")
	if ret != nil {
		db.ClearHistory(symbol, "LONG")
	}
}

func (m *LongMode) CreateBuySide(client *futures.Client, c *config.Config, symbol string, amount, lastPrice float64) {
	// 插入买单
	fmt.Println("CreateBuySide", symbol, amount, lastPrice)
	quantity := utils.RoundStepSize(amount/lastPrice, config.LotSizeMap[symbol])
	orderFilledChan := make(chan []string)
	order := m.createMarketOrder(client, symbol, strconv.FormatFloat(quantity, 'f', -1, 64), "OPEN")
	if order != nil {
		go utils.CheckOrderById(client, symbol, order.OrderID, orderFilledChan)
		values := <-orderFilledChan
		if len(values) == 3 {
			fmt.Println(symbol, values)
			//TODO 市价单查不出交易的数量只能返回平均价和总投入
			amount, _ = strconv.ParseFloat(values[0], 64)
			price, _ := strconv.ParseFloat(values[2], 64)
			quantity := amount / price
			// quantity = quantity * (1 - feeMap[pair])
			db.InsertInvestment(symbol, amount, utils.RoundStepSize(quantity, config.LotSizeMap[symbol]), price, "LONG")
		}
	}
}

func (m *LongMode) createMarketOrder(client *futures.Client, pair string, quantity string, side string) (order *futures.CreateOrderResponse) {
	var sideType futures.SideType
	var err error
	if side == "OPEN" {
		sideType = futures.SideTypeBuy
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(utils.RandStr(12)).PositionSide(futures.PositionSideTypeLong).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	} else {
		sideType = futures.SideTypeSell
		quantityF, _ := decimal.NewFromString(quantity)
		step := decimal.NewFromFloat(config.LotSizeMap[pair])
		quantity = strconv.FormatFloat(utils.RoundStepSize(quantityF.InexactFloat64(), step.InexactFloat64()), 'f', -1, 64)
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(utils.RandStr(12)).PositionSide(futures.PositionSideTypeLong).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	}
	if err != nil {
		fmt.Println("交易出错", err)
		return nil
	}
	return order
}
