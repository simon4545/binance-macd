package db

import (
	"fmt"
	"testing"
	"time"

	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
	"github.com/simon4545/binance-macd/utils"
)

var testChan chan []string

func InitCache() {
	cfg := lediscfg.NewConfigDefault()
	l, _ := ledis.Open(cfg)
	ldb, _ = l.Select(1)
}
func SetOrderCacheT(symbol string) {
	ldb.SetEX([]byte(symbol), 60*5, []byte("YES"))
}
func GetOrderCacheT(symbol string) (found bool) {
	val, err := ldb.Get([]byte(symbol))
	if err != nil || val == nil {
		return
	}
	found = true
	return
}
func TestCache(t *testing.T) {
	InitCache()
	SetOrderCacheT("BTCUSDT")
	val1 := GetOrderCacheT("BTCUSDT")
	val := GetOrderCacheT("ETHUSDT")
	fmt.Println(val1, val)
}
func TestQu(t *testing.T) {
	value := utils.RoundStepSize(64.4576, 0.01)
	fmt.Println(value)
}
func TestCheckTotalInvestment(t *testing.T) {
	CheckTotalInvestment(nil, "LONG")
}
func ch(chan []string) {
	time.Sleep(time.Second * 3)
	testChan <- []string{"order.CummulativeQuoteQuantity", "order.ExecutedQuantity"}
}
func TestCheckChannel(t *testing.T) {
	testChan = make(chan []string)
	ch(testChan)
	fmt.Println(<-testChan)
}
