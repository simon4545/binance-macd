package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
	"github.com/simon4545/binance-macd/config"
	"github.com/tidwall/gjson"
)

var SecondsPerUnit map[string]int = map[string]int{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}

func RoundStepSize(quantity float64, step_size float64) float64 {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size))).InexactFloat64()
}

func RoundStepSizeDecimal(quantity float64, step_size float64) decimal.Decimal {
	quantityD := decimal.NewFromFloat(quantity)
	return quantityD.Sub(quantityD.Mod(decimal.NewFromFloat(step_size)))
}

func GetBalance(client *futures.Client, token string) float64 {
	res, err := client.NewGetBalanceService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1
	}
	balance := 0.0
	for _, s := range res {
		if s.Asset == token {
			balance, _ = strconv.ParseFloat(s.AvailableBalance, 64)
		}
	}
	return balance
}

func GetSymbolInfo(client *futures.Client) {
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		print("Error fetching exchange info:", err)
		os.Exit(1)
	}
	for _, s := range info.Symbols {
		// if strings.HasSuffix(s.Symbol, "USDT") {
		if s.Status == string(futures.SymbolStatusTypeTrading) {
			lotSizeFilter := s.LotSizeFilter()
			quantityTickSize, _ := strconv.ParseFloat(lotSizeFilter.StepSize, 64)
			config.LotSizeMap[s.Symbol] = quantityTickSize
			priceFilter := s.PriceFilter()
			priceTickSize, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
			config.PriceFilterMap[s.Symbol] = priceTickSize
		}
		// return
		// }
	}
}
func GetFeeInfo(client *futures.Client, symbols []string) {
	for _, s := range symbols {
		if config.FeeMap[s] != 0 {
			continue
		}
		rate, err := client.NewCommissionRateService().Symbol(s).Do(context.Background())
		if err != nil {
			print("Error fetching trade fee:", err)
			os.Exit(1)
		}
		fee, _ := strconv.ParseFloat(rate.TakerCommissionRate, 64)
		config.FeeMap[s] = fee
	}
}
func Crossover(a, b []float64) bool {
	return a[len(a)-2] < b[len(b)-2] && a[len(a)-1] >= b[len(b)-1]
}

func Crossdown(a, b []float64) bool {
	return a[len(a)-2] >= b[len(b)-2] && a[len(a)-1] < b[len(b)-1]
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStr(prefix string, length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	str := fmt.Sprintf("SIM-%s-%s", prefix, string(b))
	return str
}

func CheckOrderById(client *futures.Client, pair string, orderId int64, orderFilledChan chan []string) {
	var order *futures.Order
	var err error
	for {
		order, err = client.NewGetOrderService().Symbol(pair).OrderID(orderId).Do(context.Background())
		if err != nil {
			fmt.Println("GetOrderById::error::", err)
		}
		if order.Status == futures.OrderStatusTypeFilled {
			break
		}
		time.Sleep(time.Second * 1)
	}
	fmt.Println(order)
	orderFilledChan <- []string{order.CumQuote, order.CumQuantity, order.AvgPrice}
}

func List(conf *config.Config, symbols *[]string) {
	if len(conf.Symbols) > 0 {
		*symbols = (*symbols)[:0]
		for k := range conf.Symbols {
			*symbols = append(*symbols, k)
		}
	} else {
		url := "https://api.binance.com/api/v3/ticker/24hr"
		// url := "https://api.binance.com/api/v3/ticker/24hr"
		response, err := http.Get(url)
		if err != nil {
			log.Println("Error making GET request:", err)
			return
		}
		defer response.Body.Close()
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			return
		}

		responseBody := string(bodyBytes)
		value := gjson.Parse(responseBody).Array()
		for _, symbol := range value {
			symbolCoin := symbol.Get("symbol").String()

			if !strings.HasSuffix(symbolCoin, "USDT") {
				continue
			}
			baseAsset := symbolCoin[:len(symbolCoin)-4]
			if strings.HasSuffix(baseAsset, "DOWN") || strings.HasSuffix(baseAsset, "UP") {
				continue
			}
			volume24h := symbol.Get("quoteVolume").Float()

			if volume24h > 5_000_000 {
				*symbols = append(*symbols, baseAsset)
			}
		}
	}
	// sort.Sort(symols)
	// sort.Slice(symbols, func(i, j int) bool {
	// 	return symbols[i].Percent > symbols[j].Percent
	// })
	// symbols = symbols[:100]
	Logln("总大小", len(*symbols))

}
func Logln(args ...interface{}) {
	out := os.Stdout
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fmt.Fprintf(out, "[%s] ", time.Now().In(loc).Format("2006-01-02 15:04:05"))
	fmt.Fprintln(out, args...)
}
