package main

import (
	"fmt"
	"testing"
)

func TestDB(t *testing.T) {
	db.Unscoped().Where("created_at <= DATETIME('now', '-1 days') ").Delete(&Cache{})
}
func TestOrder(t *testing.T) {
	a := fmt.Sprintf("%.2f", 1*0.2)
	fmt.Println(a)
	placeOrder("BTCUSDT", "BUY", 83278)
}
