package bn

import (
	"fmt"
	"slices"
	"time"
)

func CheckATR() {
	for _, k := range Symbols {
		Atrs[k] = calculateAtr(k, 30)
	}
	fmt.Println("Atr", Atrs)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for t := range ticker.C {
		for _, k := range Symbols {
			Atrs[k] = calculateAtr(k, 30)
		}
		fmt.Printf("执行任务，当前时间: %v\n", t)
	}
}

func CheckAmplitude() {
	for _, k := range Symbols {
		Amplitudes[k] = calculateSymbolAmplitude(k)
	}
	fmt.Println("amplitudes", Amplitudes)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop() // 确保在程序退出时停止 Ticker

	// 使用 for 循环监听 Ticker 的 channel
	for t := range ticker.C {
		for _, k := range Symbols {
			Amplitudes[k] = calculateSymbolAmplitude(k)
		}

		fmt.Printf("执行任务，当前时间: %v\n", t)
	}
}

func checkPriceDropRate(klines *KLine, symbol string) bool {
	recentHighs := make([]float64, 0)
	if len(klines.Close) < 287 {
		return false
	}
	for i := len(klines.High) - 287; i < len(klines.High); i++ {
		recentHighs = append(recentHighs, klines.High[i])
	}

	maxHigh := slices.Max(recentHighs)
	priceDropRate := 1 - klines.Price/maxHigh
	if Amplitudes[symbol] == 0 {
		return false
	}
	return priceDropRate > Amplitudes[symbol]
}

func checkRecentBullishCandles(klines *KLine) bool {
	for i := len(klines.Close) - 2; i < len(klines.Close); i++ {
		open := klines.Open[i]
		close := klines.Close[i]
		if close <= open {
			return false
		}
	}
	return true
}
