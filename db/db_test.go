package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/simon4545/binance-macd/functions"
)

var testChan chan []string

func TestQu(t *testing.T) {
	value := functions.RoundStepSize(64.4576, 0.01)
	fmt.Println(value)
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
