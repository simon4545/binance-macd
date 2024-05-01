package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/simon4545/binance-macd/utils"
)

func TestOpenLong(t *testing.T) {
	sideType := futures.SideTypeBuy
	order, err := client.NewCreateOrderService().Symbol("ARUSDT").NewClientOrderID(utils.RandStr("LONG", 15)).PositionSide(futures.PositionSideTypeLong).
		Side(sideType).Type(futures.OrderTypeMarket).Quantity("2").Do(context.Background(), futures.WithRecvWindow(10000))
	fmt.Println(order, err)
}
func TestOpenShort(t *testing.T) {
	sideType := futures.SideTypeSell
	order, err := client.NewCreateOrderService().Symbol("ARUSDT").NewClientOrderID(utils.RandStr("LONG", 12)).PositionSide(futures.PositionSideTypeShort).
		Side(sideType).Type(futures.OrderTypeMarket).Quantity("2").Do(context.Background(), futures.WithRecvWindow(10000))
	fmt.Println(order, err)
}
func TestClosePositionLong(t *testing.T) {
	sideType := futures.SideTypeSell
	order, err := client.NewCreateOrderService().Symbol("ARUSDT").NewClientOrderID(utils.RandStr("LONG", 12)).PositionSide(futures.PositionSideTypeLong).
		Side(sideType).Type(futures.OrderTypeTakeProfitMarket).ClosePosition(true).Do(context.Background(), futures.WithRecvWindow(10000))
	fmt.Println(order, err)
}
func TestCloseLong(t *testing.T) {
	sideType := futures.SideTypeSell
	order, err := client.NewCreateOrderService().Symbol("ARUSDT").NewClientOrderID(utils.RandStr("LONG", 12)).PositionSide(futures.PositionSideTypeLong).
		Side(sideType).Type(futures.OrderTypeMarket).Quantity("2").Do(context.Background(), futures.WithRecvWindow(10000))
	fmt.Println(order, err)
}
func TestCloseShort(t *testing.T) {
	sideType := futures.SideTypeBuy
	order, err := client.NewCreateOrderService().Symbol("ARUSDT").NewClientOrderID(utils.RandStr("LONG", 12)).PositionSide(futures.PositionSideTypeShort).
		Side(sideType).Type(futures.OrderTypeMarket).Quantity("2").Do(context.Background(), futures.WithRecvWindow(10000))
	fmt.Println(order, err)
}
