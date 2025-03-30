package main

import "testing"

func TestDB(t *testing.T) {
	db.Unscoped().Where("created_at <= DATETIME('now', '-1 days') ").Delete(&Cache{})
}
func TestOrder(t *testing.T) {
	placeOrder("BTCUSDT", "BUY", 83278)
}
