package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/shopspring/decimal"
)

var wsUserStop chan struct{}

func InitWS() {
	go wsUser(client)
	go wsUserReConnect()
}
func getUserStream(client *binance.Client) string {
	res, err := client.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return res
}

func wsUser(client *binance.Client) {
	listenKey := getUserStream(client)
	errHandler := func(err error) {
		fmt.Println("ws user error:", err)
	}
	var err error
	var doneC chan struct{}
	doneC, wsUserStop, err = binance.WsUserDataServe(listenKey, userWsHandler, errHandler)
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

func userWsHandler(event *binance.WsUserDataEvent) {
	if event.Event != "executionReport" {
		return
	}
	message := event.OrderUpdate
	if !strings.HasSuffix(message.Symbol, "USDT") {
		return
	}

	// if message.Status == "CANCELED" {

	// }
	if message.Status == "FILLED" {
		price, _ := strconv.ParseFloat(message.LatestPrice, 64)
		symbol := message.Symbol[:len(message.Symbol)-4]
		feeCost, _ := decimal.NewFromString(message.FeeCost)
		filledVolume, _ := decimal.NewFromString(message.FilledVolume)
		// gainVolume := filledVolume.Sub(feeCost).InexactFloat64()
		decimal.NewFromInt(99)
		gainVolume := filledVolume.Mul(decimal.NewFromFloat(0.999)).InexactFloat64()
		step := decimal.NewFromFloat(lotSizeMap[message.Symbol])
		gainVolume = RoundStepSizeDecimal(gainVolume, step.InexactFloat64()).InexactFloat64()
		quoteVolume, _ := strconv.ParseFloat(message.FilledQuoteVolume, 64)
		fmt.Println("订单成交", symbol, quoteVolume, gainVolume, price, feeCost.InexactFloat64())
		// quantity, _ := strconv.ParseFloat(message.Volume, 64)
		if strings.HasPrefix(message.ClientOrderId, "SIM-") && message.Side == string(binance.SideTypeBuy) {
			fmt.Println("订单成交-量化", symbol, quoteVolume, gainVolume, price)
			invest := Investment{
				Operate:   "BUY",
				Currency:  symbol,
				Amount:    quoteVolume,
				Quantity:  gainVolume,
				UnitPrice: price,
			}
			MakeDBInvestment(invest)
		}

	}
}
