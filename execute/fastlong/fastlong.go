package fastlong

import (
	"context"
	"fmt"
	"math"
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
	t := &FastLongMode{ModeName: "FASTLONG"}
	interfacer.Register(t.ModeName, t)
}

type FastLongMode struct {
	ModeName string
}

func avg(list []float64) float64 {
	var total float64 = 0
	for _, value := range list {
		total += value
	}
	return total / float64(len(list))
}
func (m *FastLongMode) Handle(client *futures.Client, c *config.Config, symbol string, lastPrice float64, closingPrices, highPrices, lowPrices []float64, volumes []float64) {
	config.OrderLocker.Lock()
	defer config.OrderLocker.Unlock()
	if len(closingPrices) < 99 {
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
	maxInLast99 := slices.Max(closingPrices[length-99 : length-1])
	maxInLast20 := slices.Max(closingPrices[length-21 : length-1])
	minInLast6 := slices.Min(closingPrices[length-7 : length-1])
	avgVolume := avg(volumes[length-7 : length-1])
	investCount := db.GetInvestmentCount(symbol, m.ModeName)
	lastInvest := db.InvestmentAvgPrice1(symbol, m.ModeName)
	sumInvestment := db.GetSumInvestment(symbol, m.ModeName)
	balance := db.GetSumInvestmentQuantity(symbol, m.ModeName)
	// atrRate := atr[0] / lastPrice
	if (lastPrice >= maxInLast20 && lastPrice < maxInLast99) && !db.GetOrderCache(symbol) {
		fmt.Println(symbol, m.ModeName, lastPrice, maxInLast20, maxInLast99, minInLast6)
		fmt.Println(symbol, "volume", volumes[length-1], avgVolume)
		volumeLargeThenAvg := volumes[length-1] > avgVolume*2.5
		if investCount == 0 && volumeLargeThenAvg {
			//检查上一轮是不是在正常范围内
			// if !m.CheckInRange(symbol, lastPrice, maxInLast99) {
			// 	fmt.Println(symbol, "FASTLONG 价格太高了，不进了")
			// 	return
			// }
			balance := utils.GetBalance(client, "USDT")
			if balance*10 < c.Symbols[symbol].Amount {
				fmt.Println(symbol, "余额不足", balance)
				return
			}
			sl := minInLast6
			//插入买单
			db.MakeLog(symbol, fmt.Sprintf("FASTLONG %s 急速上涨 价格:%f 止损%f 止盈%f",
				time.Now().Format("2006-01-02 15:04:05"),
				lastPrice, sl, lastPrice+math.Abs(lastPrice-sl)*1.5,
			))
			m.CreateBuySide(client, c, symbol, c.Symbols[symbol].Amount, lastPrice, sl)
		}
	}
	if investCount > 0 && balance > config.LotSizeMap[symbol] {
		// _atrRate := atrRate * 3.1
		// cond1 := (balance * lastPrice) <= sumInvestment*(1-_atrRate)
		cond1 := lastPrice > lastInvest.TakeProfit
		cond2 := lastPrice <= minInLast6
		if cond1 || cond2 {
			// if hits[len(hits)-2] > 0 && hits[len(hits)-1] <= 0 {
			db.MakeLog(symbol, fmt.Sprintf("FASTLONG 出场 %f %f GetSumInvestment %f GetInvestmentCount %d cond1:%t cond2:%t", lastPrice, lastInvest.TakeProfit, sumInvestment, investCount, cond1, cond2))
			m.CreateSellSide(client, c, symbol, balance)
		}
	}
	// m.SetInRange(symbol, lastPrice, maxInLast20, minInLast6)
}
func (m *FastLongMode) CheckInRange(symbol string, lastPrice, maxValue float64) bool {
	//上一轮在范围中
	// lastResult := db.GetInRange(symbol, m.ModeName)
	return lastPrice < maxValue
}
func (m *FastLongMode) SetInRange(symbol string, lastprice, upper, lower float64) {
	if lastprice > lower && lastprice < upper {
		db.SetInRange(symbol, m.ModeName, true)
		return
	}
	db.SetInRange(symbol, m.ModeName, false)
}
func (m *FastLongMode) CreateSellSide(client *futures.Client, c *config.Config, symbol string, balance float64) {
	quantity := utils.RoundStepSize(balance, config.LotSizeMap[symbol])
	fmt.Println(symbol, "quantity", quantity)
	// 插入卖单
	ret := m.createMarketOrder(client, symbol, strconv.FormatFloat(quantity, 'f', -1, 64), "CLOSE")
	if ret != nil {
		db.ClearHistory(symbol, m.ModeName)
	}
}

func (m *FastLongMode) CreateBuySide(client *futures.Client, c *config.Config, symbol string, amount, lastPrice, sl float64) {
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
			_priceF := _price.InexactFloat64()
			tp := _priceF + math.Abs(_priceF-sl)*1.5
			db.Insert(symbol, _amount.InexactFloat64(), quantity, _price.InexactFloat64(), tp, sl, m.ModeName)
		}
	}
}

func (m *FastLongMode) createMarketOrder(client *futures.Client, pair string, quantity string, side string) (order *futures.CreateOrderResponse) {
	var sideType futures.SideType
	var err error
	// 开空
	if side == "OPEN" {
		sideType = futures.SideTypeBuy
		order, err = client.NewCreateOrderService().Symbol(pair).NewClientOrderID(utils.RandStr(12)).PositionSide(futures.PositionSideTypeLong).
			Side(sideType).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background(), futures.WithRecvWindow(10000))
	} else {
		// 平空
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
