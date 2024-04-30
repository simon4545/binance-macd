package fastshort

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/config"
	"github.com/simon4545/binance-macd/db"
	"github.com/simon4545/binance-macd/interfacer"
	"github.com/simon4545/binance-macd/utils"
)

func init() {
	t := &FastShortMode{}
	interfacer.Register("FASTSHORT", t)
}

type FastShortMode struct {
}

func avg(list []float64) float64 {
	var total float64 = 0
	for _, value := range list {
		total += value
	}
	return total / float64(len(list))
}
func (m *FastShortMode) Handle(client *futures.Client, c *config.Config, symbol string, lastPrice float64, closingPrices, highPrices, lowPrices []float64, volumes []float64) {
	config.OrderLocker.Lock()
	defer config.OrderLocker.Unlock()
	if len(closingPrices) < 30 {
		return
	}
	atr, exists := config.AtrMap.Get(symbol)
	if !exists {
		return
	}
	//TODO 如果FDUSD交易对，fee本身就是0，这里需要做一次单独处理
	if config.LotSizeMap[symbol] == 0 || atr[0] == 0 || config.FeeMap[symbol] == 0 {
		fmt.Println("没有拿到精度")
		return
	}
	// _, _, hits := talib.Macd(closingPrices, 12, 26, 9)
	length := len(closingPrices)
	minInLast20 := slices.Min(closingPrices[length-20 : length-1])
	minInLast5 := slices.Min(closingPrices[length-6 : length-1])
	maxInLast3 := slices.Max(closingPrices[length-4 : length-1])
	investCount := db.GetInvestmentCount(symbol, "FASTSHORT")
	sumInvestment := db.GetSumInvestment(symbol, "FASTSHORT")
	balance := db.GetSumInvestmentQuantity(symbol, "FASTSHORT")
	atrRate := atr[0] / lastPrice
	fmt.Println(symbol, atr, atrRate)
	if lastPrice <= minInLast5 && lastPrice > minInLast20 {
		if lastPrice <= atr[2]*1.05 {
			fmt.Println("价格太低了，不空了")
			return
		}
		volumeLargeThenAvg := closingPrices[length-1] > avg(volumes[length-6:length-1])*1.5
		if investCount == 0 && volumeLargeThenAvg {
			balance := utils.GetBalance(client, "USDT")
			if balance*10 < c.Symbols[symbol].Amount {
				fmt.Println(symbol, "余额不足", balance)
				return
			}
			//插入买单
			db.MakeLog(symbol, fmt.Sprintf("FASTSHORT 当前时间 %s 出现急速下跌 价格:%f ",
				time.Now().Format("2006-01-02 15:04:05"),
				lastPrice,
			))
			m.CreateBuySide(client, c, symbol, c.Symbols[symbol].Amount, lastPrice)
		}
	}
	if investCount > 0 && balance > config.LotSizeMap[symbol] {
		_atrRate := atrRate * 3.1
		cond1 := (balance * lastPrice) <= sumInvestment*(1-_atrRate)
		cond2 := lastPrice >= maxInLast3
		if cond1 || cond2 {
			// if hits[len(hits)-2] > 0 && hits[len(hits)-1] <= 0 {
			db.MakeLog(symbol, fmt.Sprintf("FASTSHORT 出现回升 GetSumInvestment %f GetInvestmentCount %d cond1:%t cond2:%t", sumInvestment, investCount, cond1, cond2))
			m.CreateSellSide(client, c, symbol, balance)
		}
	}
}

func (m *FastShortMode) CreateSellSide(client *futures.Client, c *config.Config, symbol string, balance float64) {
	quantity := utils.RoundStepSize(balance, config.LotSizeMap[symbol])
	fmt.Println(symbol, "quantity", quantity)
	// 插入卖单
	ret := m.createMarketOrder(client, symbol, strconv.FormatFloat(quantity, 'f', -1, 64), "CLOSE")
	if ret != nil {
		db.ClearHistory(symbol, "FASTSHORT")
	}
}

func (m *FastShortMode) CreateBuySide(client *futures.Client, c *config.Config, symbol string, amount, lastPrice float64) {
	// 插入买单
	fmt.Println("CreateBuySide", symbol, amount, lastPrice)
	damount := decimal.NewFromFloat(amount)
	dlastPrice := decimal.NewFromFloat(lastPrice)
	quantity := utils.RoundStepSize(damount.Div(dlastPrice).InexactFloat64(), config.LotSizeMap[symbol])
	orderFilledChan := make(chan []string)
	order := m.createMarketOrder(client, symbol, strconv.FormatFloat(quantity, 'f', -1, 64), "OPEN")
	if order != nil {
		go utils.CheckOrderById(client, symbol, order.OrderID, orderFilledChan)
		values := <-orderFilledChan
		if len(values) == 3 {
			fmt.Println(symbol, values)
			//TODO 市价单查不出交易的数量只能返回平均价和总投入
			_amount, _ := decimal.NewFromString(values[0])
			_price, _ := decimal.NewFromString(values[2])
			// quantity := _amount.Div(_price).InexactFloat64()
			// quantity = quantity * (1 - feeMap[pair])
			db.InsertInvestment(symbol, _amount.InexactFloat64(), quantity, _price.InexactFloat64(), "FASTSHORT")
		}
	}
}

func (m *FastShortMode) createMarketOrder(client *futures.Client, pair string, quantity string, side string) (order *futures.CreateOrderResponse) {
	var sideType futures.SideType
	var err error
	// 开空
	if side == "OPEN" {
		sideType = futures.SideTypeSell
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(utils.RandStr(12)).PositionSide(futures.PositionSideTypeShort).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	} else {
		// 平空
		sideType = futures.SideTypeBuy
		quantityF, _ := decimal.NewFromString(quantity)
		step := decimal.NewFromFloat(config.LotSizeMap[pair])
		quantity = strconv.FormatFloat(utils.RoundStepSize(quantityF.InexactFloat64(), step.InexactFloat64()), 'f', -1, 64)
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(utils.RandStr(12)).PositionSide(futures.PositionSideTypeShort).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	}
	if err != nil {
		fmt.Println("交易出错", err)
		return nil
	}
	return order
}
