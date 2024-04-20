package main

import (
	"fmt"
	"testing"
	"time"
)

var testChan chan []string

func TestQu(t *testing.T) {
	value := RoundStepSize(64.4576, 0.01)
	fmt.Println(value)
}
func TestCheckTotalInvestment(t *testing.T) {
	CheckTotalInvestment()
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
