package bn

import (
	"time"

	"github.com/simon4545/binance-macd/db"
	"github.com/simon4545/binance-macd/functions"
)

func getInvestmentCount(invests []db.Investment) int {
	return len(invests)
}

func getSumInvestment(invests []db.Investment) (sum float64) {
	for _, item := range invests {
		sum += item.Amount
	}
	return
}

func getSumInvestmentQuantity(invests []db.Investment) (sum float64) {
	for _, item := range invests {
		sum += item.Quantity
	}
	return
}
func recentInvestment(invests []db.Investment) (invest *db.Investment) {
	if len(invests) == 0 {
		return nil
	}
	return &invests[len(invests)-1]
}
func recentInvestmentPrice(invests []db.Investment, period string) (price float64) {
	if len(invests) == 0 {
		return -1
	}
	intPeriod := functions.ConvertToSeconds(period)
	current := time.Now().Add(-time.Duration(intPeriod*15) * time.Second)
	invest := invests[len(invests)-1]
	if invest.CreatedAt.Before(current) {
		return -1
	}
	return invest.UnitPrice
}
