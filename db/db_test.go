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
var testdb *ledis.DB

func InitCache() {
	cfg := lediscfg.NewConfigDefault()
	l, _ := ledis.Open(cfg)
	testdb, _ = l.Select(1)
}
func SetOrderCacheT(symbol string) {
	testdb.SetEX([]byte(symbol), 60*5, []byte("YES"))
}
func GetOrderCacheT(symbol string) (found bool) {
	val, err := testdb.Get([]byte(symbol))
	if err != nil || val == nil {
		return
	}
	found = true
	return
}
func SetInRange1(symbol, mode string, inrange bool) {
	if inrange {
		testdb.SetEX([]byte(symbol+mode), 20, []byte{1})
	} else {
		testdb.SetEX([]byte(symbol+mode), 20, []byte{0})
	}
}
func GetInRange1(symbol, mode string) (inrange bool) {
	val, err := testdb.Get([]byte(symbol + mode))
	if err != nil || val == nil {
		return
	}
	in := val[0]
	if in == 1 {
		inrange = true
	}
	return
}
func TestRange(t *testing.T) {
	InitCache()
	SetInRange1("BTCUSDT", "FASTSHORT", false)
	val1 := GetInRange1("BTCUSDT", "FASTSHORT")
	fmt.Println(val1)
}
func TestCache(t *testing.T) {
	InitCache()
	SetOrderCacheT("BTCUSDT")
	val1 := GetOrderCacheT("BTCUSDT")
	val := GetOrderCacheT("ETHUSDT")
	fmt.Println(val1, val)
}
func TestCache1(t *testing.T) {
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
