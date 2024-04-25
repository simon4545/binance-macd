package main

// Type
// MARKET 市价单
// LIMIT 限价单
// STOP 止损单
// TAKE_PROFIT 止盈单
// LIQUIDATION 强平单
// ExecutionType
// NEW
// CANCELED 已撤
// CALCULATED 订单ADL或爆仓
// EXPIRED 订单失效
// TRADE 交易
// AMENDMENT 订单修改
import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/db"
	"github.com/simon4545/binance-macd/interfacer"
)

var wsUserStop chan struct{}

func InitWS() {
	go wsUser(client)
	go wsUserReConnect()
}
func getUserStream(client *futures.Client) string {
	res, err := client.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return res
}

func wsUser(client *futures.Client) {
	listenKey := getUserStream(client)
	errHandler := func(err error) {
		fmt.Println("ws user error:", err)
	}
	var err error
	var doneC chan struct{}
	doneC, wsUserStop, err = futures.WsUserDataServe(listenKey, userWsHandler, errHandler)
	if err != nil {
		fmt.Println("ws user error:", err)
		return
	}
	<-doneC
	time.Sleep(3 * time.Second)
	wsUser(client)
}

func wsUserReConnect() {
	for {
		time.Sleep(55 * time.Minute)
		fmt.Println("1hour reconnect wsUser")
		wsUserStop <- struct{}{}
	}
}

func userWsHandler(event *futures.WsUserDataEvent) {
	if event.Event != "ORDER_TRADE_UPDATE" {
		return
	}
	message := event.OrderTradeUpdate
	if !strings.HasSuffix(message.Symbol, "USDT") {
		return
	}

	// if message.Status == "CANCELED" {

	// }
	if message.Status == "FILLED" && message.Side == futures.SideTypeSell {
		price, _ := strconv.ParseFloat(message.LastFilledPrice, 64)
		symbol := message.Symbol[:len(message.Symbol)-4]
		feeCost, _ := decimal.NewFromString(message.Commission)
		quoteVolume, _ := strconv.ParseFloat(message.AccumulatedFilledQty, 64)
		fmt.Println("订单成交", symbol, quoteVolume, price, feeCost.InexactFloat64(), "完整信息", message)
		// quantity, _ := strconv.ParseFloat(message.Volume, 64)
		// if strings.HasPrefix(message.ClientOrderID, "SIM-") && message.Side == futures.SideTypeBuy {
		// 	orderFilledChan <- []string{order.CumQuote, message.AccumulatedFilledQty, message.LastFilledPrice}
		// }

		fmt.Println("订单成交-量化", symbol, quoteVolume, price, message)
		// if strings.HasPrefix(message.ClientOrderID, "SIM-") {
		db.ClearHistory(symbol, string(message.PositionSide))
		// }
		if message.Type == "LIQUIDATION" {
			fmt.Println("这是强平单空，要立即补仓", message.ExecutionType)
			symbol, found := strings.CutSuffix(message.Symbol, "USDT")
			if found && slices.Contains(symbols, symbol) {
				// 补仓
				excutor := interfacer.Create(string(message.PositionSide), client)
				excutor.CreateBuySide(client, conf, symbol, message.Symbol, quoteVolume*price, price)
			}
		}

	}
}
