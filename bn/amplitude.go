package bn

import (
	"context"
	"sort"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/markcheno/go-talib"
)

func calculateSymbolAmplitude(symbol string) float64 {
	amplitudeAvg, _ := calculateAmplitudeAverage(symbol, 30)
	return amplitudeAvg
}

// 计算ATR
func calculateAtr(symbol string, limit int) float64 {
	klines, err := getKlines(symbol, "4h", limit)
	if err != nil {
		return 0
	}
	var closes []float64
	var highs []float64
	var lows []float64
	// var opens []float64
	for _, kline := range klines {
		// open, _ := strconv.ParseFloat(kline.Open, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		close, _ := strconv.ParseFloat(kline.Close, 64)
		closes = append(closes, close)
		highs = append(highs, high)
		lows = append(lows, low)
		// opens = append(opens, open)
	}
	atrs := talib.Atr(highs, lows, closes, 14)
	return atrs[len(atrs)-1]
}

// 计算振幅平均值
func calculateAmplitudeAverage(symbol string, limit int) (float64, error) {
	klines, err := getKlines(symbol, "1d", limit)
	if err != nil {
		return 0, err
	}

	amplitudes := make([]float64, 0)
	for _, kline := range klines {
		open, _ := strconv.ParseFloat(kline.Open, 64)
		close, _ := strconv.ParseFloat(kline.Close, 64)
		if close >= open {
			continue
		}
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		amplitude := high - low
		amplitude /= high
		amplitudes = append(amplitudes, amplitude)
	}

	top3Amplitudes := getTopNValues(amplitudes, 5)

	sum := 0.0
	for _, amplitude := range top3Amplitudes {
		sum += amplitude
	}
	average := sum / float64(len(top3Amplitudes))

	return average * 0.9, nil
}

func getTopNValues(values []float64, n int) []float64 {
	// 复制切片以避免修改原始数据
	copiedValues := make([]float64, len(values))
	copy(copiedValues, values)

	// 排序（降序）
	sort.Sort(sort.Reverse(sort.Float64Slice(copiedValues)))

	// 返回前N个值
	if n > len(copiedValues) {
		n = len(copiedValues)
	}
	return copiedValues[1:n]
}

//	func calculateATR(klines []*binance.Kline) float64 {
//		var sum float64
//		for i := 1; i < len(klines); i++ {
//			high, _ := strconv.ParseFloat(klines[i].High, 64)
//			low, _ := strconv.ParseFloat(klines[i].Low, 64)
//			prevClose, _ := strconv.ParseFloat(klines[i-1].Close, 64)
//			tr := math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
//			sum += tr
//		}
//		return sum / float64(len(klines)-1)
//	}
func getKlines(symbol, interval string, limit int) ([]*futures.Kline, error) {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return klines, nil
}
