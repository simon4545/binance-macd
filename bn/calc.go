package bn

import (
	"fmt"
	"slices"
	"time"

	"github.com/simon4545/binance-macd/configuration"
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
func FirstInvestment(invests []db.Investment) (invest *db.Investment) {
	if len(invests) == 0 {
		return nil
	}
	return &invests[0]
}
func RecentInvestment(invests []db.Investment) (invest *db.Investment) {
	if len(invests) == 0 {
		return nil
	}
	return &invests[len(invests)-1]
}
func TodayInvestment(invests []db.Investment) (count int) {
	if len(invests) == 0 {
		return 0
	}
	now := time.Now()
	for _, v := range invests {
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if v.CreatedAt.After(startOfDay) {
			count++
		}
	}
	return
}
func recentInvestmentPrice(invests []db.Investment, period string, multi int) (price float64) {
	if len(invests) == 0 {
		return -1
	}
	intPeriod := functions.ConvertToSeconds(period)
	current := time.Now().Add(-time.Duration(intPeriod*multi) * time.Second)
	invest := invests[len(invests)-1]
	if invest.CreatedAt.Before(current) {
		return -1
	}
	return invest.UnitPrice
}
func CheckAmplitude() {
	for k, _ := range c.Symbols {
		Amplitudes[k] = calculateSymbolAmplitude(k)
	}
	fmt.Println("amplitudes", Amplitudes)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop() // 确保在程序退出时停止 Ticker

	// 使用 for 循环监听 Ticker 的 channel
	for t := range ticker.C {
		for k, _ := range c.Symbols {
			Amplitudes[k] = calculateSymbolAmplitude(k)
		}

		fmt.Printf("执行任务，当前时间: %v\n", t)
	}
}

func checkPriceDropRate(klines *configuration.KLine, symbol, period string) bool {
	// 动态计算过去 24 小时的 K 线数量
	count := calculateKLineCountFor24h(period)
	if count <= 0 {
		return false
	}
	// recentHighs := make([]float64, 0)
	if len(klines.Close) < count {
		return false
	}
	// for i := len(klines.High) - 287; i < len(klines.High); i++ {
	// 	recentHighs = append(recentHighs, klines.High[i])
	// }
	// 取最近 count 根 K 线的最高价
	startIdx := len(klines.High) - count
	if startIdx < 0 {
		startIdx = 0
	}
	recentHighs := klines.High[startIdx:]
	maxHigh := slices.Max(recentHighs)
	priceDropRate := 1 - klines.Price/maxHigh
	if Amplitudes[symbol] == 0 {
		return false
	}
	return priceDropRate > Amplitudes[symbol]
}

func checkRecentBullishCandles(klines *configuration.KLine) bool {
	for i := len(klines.Close) - 2; i < len(klines.Close); i++ {
		open := klines.Open[i]
		close := klines.Close[i]
		if close <= open {
			return false
		}
	}
	return true
}
func CalcSpacing(invests []db.Investment, rate float64) (spacing float64) {
	first := FirstInvestment(invests)
	if first == nil {
		panic("data error")
	}
	return (first.UnitPrice / 100) * rate * 100
}

// calculateKLineCountFor24h 使用 ConvertToSeconds 计算 24 小时包含多少根 K 线
func calculateKLineCountFor24h(period string) int {
	secondsPerBar := functions.ConvertToSeconds(period)
	if secondsPerBar <= 0 {
		return 0
	}

	// 24 小时 = 24 * 60 * 60 秒
	totalSeconds := 24 * 60 * 60
	return totalSeconds / secondsPerBar
}
