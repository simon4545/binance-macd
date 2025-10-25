package functions

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/markcheno/go-talib"
	"github.com/remeh/sizedwaitgroup"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/configuration"
)

var atrMap map[string]float64

func SplitSymbol(symbol string) (string, string) {
	if strings.HasSuffix(symbol, "USDT") {
		return symbol[:len(symbol)-4], symbol[len(symbol)-4:]
	} else {
		return symbol[:len(symbol)-5], symbol[len(symbol)-5:]
	}
}
func RoundStepSize(quantity float64, step_size float64) float64 {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size))).InexactFloat64()
}

func RoundStepSizeDecimal(quantity float64, step_size float64) decimal.Decimal {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size)))
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStr(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	str := fmt.Sprintf("SIM-%s", string(b))
	return str
}

func Crossover(a, b []float64) bool {
	return a[len(a)-2] < b[len(b)-2] && a[len(a)-1] >= b[len(b)-1]
}

func Crossdown(a, b []float64) bool {
	return a[len(a)-2] >= b[len(b)-2] && a[len(a)-1] < b[len(b)-1]
}
func SuperTreand(assetInfo *configuration.KLine) (float64, float64) {
	max := talib.Max(assetInfo.High, 4)
	min := talib.Min(assetInfo.Low, 4)
	return max[len(max)-1], min[len(min)-1]
}
func checkAtr(client *binance.Client, symbol string, config *configuration.Config) {
	klines, err := client.NewKlinesService().Symbol(symbol + "USDT").Interval(config.Symbols[symbol].Period).Limit(100).Do(context.Background())
	if err != nil {
		print(err)
		return
	}
	closingPrices := []float64{}
	highPrices := []float64{}
	lowPrices := []float64{}
	for _, kline := range klines {
		close, _ := strconv.ParseFloat(kline.Close, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		closingPrices = append(closingPrices, close)
		highPrices = append(highPrices, high)
		lowPrices = append(lowPrices, low)
	}
	atr := talib.Atr(highPrices, lowPrices, closingPrices, 12)
	atrMap[symbol] = atr[len(atr)-1]
	// fmt.Println(symbol, "atr", atrMap[symbol])
}

func CheckAtr(client *binance.Client, symbols []string, config *configuration.Config) {
	for {
		fmt.Println(time.Now(), "开启新的一轮")
		for _, s := range symbols {
			checkAtr(client, s, config)
			time.Sleep(time.Millisecond * 100)
		}
		time.Sleep(time.Second * 20)
	}
}

func CheckCross(client *binance.Client, symbol string, config *configuration.Config, handle func(symbol string, lastPrice float64, closingPrices, highPrices, lowPrices []float64)) {
	// defer time.Sleep(4 * time.Second)
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(config.Symbols[symbol].Period).Limit(100).Do(context.Background())
	if err != nil {
		print(err)
		return
	}
	closingPrices := []float64{}
	highPrices := []float64{}
	lowPrices := []float64{}
	for _, kline := range klines {
		if kline.CloseTime > time.Now().Unix()*1000 {
			continue
		}
		close, _ := strconv.ParseFloat(kline.Close, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		closingPrices = append(closingPrices, close)
		highPrices = append(highPrices, high)
		lowPrices = append(lowPrices, low)
	}
	lastPrice, _ := strconv.ParseFloat(klines[len(klines)-1].Close, 64)
	handle(symbol, lastPrice, closingPrices, highPrices, lowPrices)
}

func CheckCrossLoop(client *binance.Client, symbols []string, config *configuration.Config, handle func(symbol string, lastPrice float64, closingPrices, highPrices, lowPrices []float64)) {
	for {
		fmt.Println(time.Now(), "开启新的一轮", config)
		swg := sizedwaitgroup.New(4)
		for _, sym := range symbols {
			swg.Add()
			go func(sym string) {
				defer swg.Done()
				CheckCross(client, sym, config, handle)
				time.Sleep(time.Millisecond * 100)
			}(sym)
		}
		swg.Wait()
		time.Sleep(time.Second * 6)
	}
}

func ConvertToSeconds(s string) int {
	sValue := strings.Split(s, "")
	sNewStr := sValue[:len(sValue)-1]
	i, err := strconv.Atoi(strings.Join(sNewStr, ""))
	if err != nil {
		panic(err)
	}
	return i * configuration.SecondsPerUnit[sValue[len(sValue)-1]]
}
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
