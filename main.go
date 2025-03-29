package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	longPriod  = "30m"
	shortPriod = "5m"
)

type Config struct {
	APIKey     string             `yaml:"api_key"`
	APISecret  string             `yaml:"api_secret"`
	Bet        map[string]float64 `yaml:"bet"`
	AtrMultier map[string]float64 `yaml:"atrmultier"`
}

var (
	cfg            Config
	atrValues      = make(map[string]float64)
	positonOrder   = map[string]bool{}
	db             *gorm.DB
	client         *futures.Client
	delock         sync.Mutex
	LotSizeMap     = make(map[string]float64)
	PriceFilterMap = make(map[string]float64)
	FeeMap         = make(map[string]float64)
	wsUserStop     chan struct{}
)

type Cache struct {
	gorm.Model
	Key   string
	Value bool
}

func init() {
	// Load config from YAML
	configFile, err := os.ReadFile("w.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}
	err = yaml.Unmarshal(configFile, &cfg)
	if err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}
	symbols := keys(cfg.Bet)
	for _, symbol := range symbols {
		if cfg.AtrMultier[symbol] == 0.0 {
			cfg.AtrMultier[symbol] = 1.0
		}
	}
	// Initialize SQLite database
	var errDB error
	db, errDB = gorm.Open(sqlite.Open("cache.db?_loc=Asia/Shanghai"), &gorm.Config{})
	if errDB != nil {
		log.Fatalf("Error opening database: %v", errDB)
	}
	db.AutoMigrate(&Cache{})

	// Initialize Binance client
	client = futures.NewClient(cfg.APIKey, cfg.APISecret)
	futures.WebsocketKeepalive = true
	// binance.WebsocketTimeout = time.Second * 1
	//校准时间
	client.NewSetServerTimeService().Do(context.Background())
}

func GetSymbolInfo(client *futures.Client) {
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		print("Error fetching exchange info:", err)
		os.Exit(1)
	}
	for _, s := range info.Symbols {
		// if strings.HasSuffix(s.Symbol, "USDT") {
		lotSizeFilter := s.LotSizeFilter()
		quantityTickSize, _ := strconv.ParseFloat(lotSizeFilter.StepSize, 64)
		LotSizeMap[s.Symbol] = quantityTickSize
		priceFilter := s.PriceFilter()
		priceTickSize, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
		PriceFilterMap[s.Symbol] = priceTickSize
		// return
		// }
	}

	// rate, err := client.NewTradeFeeService().Do(context.Background())
	// if err != nil {
	// 	print("Error fetching trade fee:", err)
	// 	os.Exit(1)
	// }
	// for _, s := range rate {
	// 	fee, _ := strconv.ParseFloat(s.TakerCommission, 64)
	// 	FeeMap[s.Symbol] = fee
	// }
}

func main() {
	go wsUser(client)
	go wsUserReConnect()
	go UpdateATR()
	updateATR()
	GetSymbolInfo(client)
	listenWebSocket()
}
func setAtr(symbol string, atr float64, lastClose float64) {
	delock.Lock()
	defer delock.Unlock()
	atrValues[symbol] = atr * cfg.AtrMultier[symbol]
	atrValues[symbol] = math.Max(0.01*lastClose, atrValues[symbol])
	atrPercent := atrValues[symbol] * 100 / lastClose
	log.Printf("%s ATR: %.3f %.2f%%\n", symbol, atrValues[symbol], atrPercent)
}
func fetchATR(symbol string) {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(longPriod).Limit(16).Do(context.Background())
	if err != nil {
		log.Printf("Error fetching ATR for %s: %v", symbol, err)
		return
	}

	var trs []float64
	for i := 1; i < len(klines); i++ {
		if klines[i].CloseTime > time.Now().UnixMilli() {
			continue
		}
		high := parseFloat(klines[i].High)
		low := parseFloat(klines[i].Low)
		closePrev := parseFloat(klines[i-1].Close)
		trs = append(trs, math.Max(high-low, math.Max(math.Abs(high-closePrev), math.Abs(low-closePrev))))
	}

	atr := sum(trs) / float64(len(trs))
	lastClose := parseFloat(klines[len(klines)-1].Close)
	setAtr(symbol, atr, lastClose)
}

func hasPumped(symbol string, side string, closePrice float64) bool {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(shortPriod).Limit(60).Do(context.Background())
	if err != nil {
		log.Printf("Error fetching ATR for %s: %v", symbol, err)
		return true
	}
	if len(klines) < 21 {
		return true
	}
	var closes []float64
	var highs []float64
	var lows []float64
	for i := 0; i < len(klines); i++ {
		if klines[i].CloseTime > time.Now().UnixMilli() {
			continue
		}
		close := parseFloat(klines[i].Close)
		high := parseFloat(klines[i].High)
		low := parseFloat(klines[i].Low)
		closes = append(closes, close)
		highs = append(highs, high)
		lows = append(lows, low)
	}
	if side == "BUY" {
		lowest := slices.Min(closes[len(closes)-8:])
		if math.Abs(closePrice-lowest) > atrValues[symbol]*3 {
			return true
		}
	}
	if side == "SELL" {
		highest := slices.Max(closes[len(closes)-8:])
		if math.Abs(highest-closePrice) > atrValues[symbol]*3 {
			return true
		}
	}
	ema20 := CalculateEMA(closes, 20)
	for i := len(ema20) - 8; i < len(ema20); i++ {
		x := ema20[i]
		if x <= highs[i] && x >= lows[i] {
			return false
		}
	}
	return true
}

func listenWebSocket() {
	symbols := keys(cfg.Bet)
	symbolsWithInterval := make(map[string]string)
	for _, symbol := range symbols {
		symbolsWithInterval[symbol] = shortPriod
	}
	doneC, stopC, err := futures.WsCombinedKlineServe(symbolsWithInterval, func(event *futures.WsKlineEvent) {
		if !event.Kline.IsFinal {
			return
		}
		openPrice := parseFloat(event.Kline.Open)
		closePrice := parseFloat(event.Kline.Close)
		priceChange := closePrice - openPrice
		atrValue := math.Max(0.012*openPrice, atrValues[event.Symbol])
		if atrValue > 0 && math.Abs(priceChange) > atrValue && !positonOrder[event.Symbol] {
			direction := "BUY"
			if priceChange < 0 {
				direction = "SELL"
			}
			log.Printf("开仓信号: %s %s %v\n", event.Symbol, direction, positonOrder[event.Symbol])
			positonOrder[event.Symbol] = true
			placeOrder(event.Symbol, direction, closePrice)
		}
	}, func(err error) {
		log.Printf("WebSocket error: %v", err)
	})
	if err != nil {
		log.Printf("WebSocket error: %v", err)
		return
	}
	defer close(stopC)
	<-doneC
}

func placeOrder(symbol, side string, price float64) {
	var cache Cache
	var hourCount int64
	if err := db.Where("key = ? AND created_at >= DATETIME('now', '-1 hours') ", symbol).First(&cache).Error; err == nil {
		return
	}

	db.Where("created_at >= DATETIME('now', '-1 hours') ").Count(&hourCount)
	if hourCount > 4 {
		return
	}

	if hasPumped(symbol, side, price) {
		fmt.Println(symbol, "价格已不正常不处理")
		return
	}
	db.Create(&Cache{Key: symbol, Value: true})
	quantity := roundStepSize(cfg.Bet[symbol]/price, LotSizeMap[symbol])
	order, err := client.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(side)).NewClientOrderID(RandStr("SIM-", 12)).
		Type(futures.OrderTypeMarket).Quantity(fmt.Sprintf("%f", quantity)).NewOrderResponseType(futures.NewOrderRespTypeRESULT).Do(context.Background())
	if err != nil {
		log.Printf("Error placing order: %v", err)
		return
	}
	log.Printf("已开仓: %s %s, %+v\n", symbol, side, order)

	price = parseFloat(order.AvgPrice)
	stopLoss := price * 0.975
	_side := "SELL"
	if side == "SELL" {
		stopLoss = price * 1.025
		_side = "BUY"
	}
	stopLoss = roundStepSize(stopLoss, PriceFilterMap[symbol])
	_, err = client.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(_side)).Type(futures.OrderTypeStopMarket).ClosePosition(true).
		NewClientOrderID(RandStr("SIMC-", 12)).StopPrice(fmt.Sprintf("%f", stopLoss)).Quantity(fmt.Sprintf("%f", quantity)).Do(context.Background())
	if err != nil {
		log.Printf("Error setting stop loss: %v", err)
		return
	}
	log.Printf("止损设置: %f %f\n", price, stopLoss)

	takeProfit := price + 0.3*atrValues[symbol]
	if side == "SELL" {
		takeProfit = price - 0.3*atrValues[symbol]
	}
	takeProfit = roundStepSize(takeProfit, PriceFilterMap[symbol])
	_, err = client.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(_side)).Type(futures.OrderTypeTrailingStopMarket).ClosePosition(true).
		NewClientOrderID(RandStr("SIMC-", 12)).ActivationPrice(fmt.Sprintf("%f", takeProfit)).CallbackRate("1").Quantity(fmt.Sprintf("%f", quantity)).Do(context.Background())
	if err != nil {
		log.Printf("Error setting take profit: %v", err)
		return
	}
	log.Printf("止盈设置: %f %f %f\n", price, atrValues[symbol], takeProfit)
}
func updateATR() {
	symbols := keys(cfg.Bet)
	for _, symbol := range symbols {
		fetchATR(symbol)
	}
}
func UpdateATR() {
	for {
		time.Sleep(5 * time.Minute)
		updateATR()
		// symbols := keys(cfg.Bet)
		// swg := sizedwaitgroup.New(4)
		// for _, s := range symbols {
		// 	swg.Add()
		// 	go func(pair string) {
		// 		defer swg.Done()
		// 		fetchATR(pair)
		// 		time.Sleep(100 * time.Millisecond)
		// 	}(s)
		// }
		// swg.Wait()
		// for _, symbol := range symbols {
		// 	fetchATR(symbol)
		// }
	}
}

func roundStepSize(quantity, stepSize float64) float64 {
	return math.Floor(quantity/stepSize) * stepSize
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func sum(arr []float64) float64 {
	var sum float64
	for _, v := range arr {
		sum += v
	}
	return sum
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStr(prefix string, length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	str := fmt.Sprintf("%s%s", prefix, string(b))
	return str
}
func keys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getUserStream(client *futures.Client) string {
	res, err := client.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return res
}

func wsUser(client *futures.Client) {
	listenKey := getUserStream(client)
	errHandler := func(err error) {
		fmt.Println("ws1 user error:", err)
	}
	var err error
	var doneC chan struct{}
	doneC, wsUserStop, err = futures.WsUserDataServe(listenKey, userWsHandler, errHandler)
	if err != nil {
		fmt.Println("ws2 user error:", err)
		return
	}
	<-doneC
	time.Sleep(3 * time.Second)
	wsUser(client)
}

func wsUserReConnect() {
	for {
		time.Sleep(55 * time.Minute)
		fmt.Println("1hour reconnect wsUser")
		wsUserStop <- struct{}{}
	}
}

func userWsHandler(event *futures.WsUserDataEvent) {
	if event.Event != futures.UserDataEventTypeOrderTradeUpdate {
		return
	}
	message := event.OrderTradeUpdate
	if message.Status == "FILLED" {
		// quantity, _ := strconv.ParseFloat(message.Volume, 64)
		symbol := message.Symbol
		if strings.HasPrefix(message.ClientOrderID, "SIMC-") {
			positonOrder[symbol] = false
			client.NewCancelAllOpenOrdersService().Symbol(symbol).Do(context.Background())
		}
	}
}

// CalculateEMA 计算EMA
// prices: 价格序列，从旧到新排列
// period: 周期，如20
func CalculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}

	ema := make([]float64, len(prices))
	multiplier := 2.0 / float64(period+1)

	// 第一个EMA值是简单移动平均(SMA)
	sma := 0.0
	for i := 0; i < period; i++ {
		sma += prices[i]
	}
	sma /= float64(period)
	ema[period-1] = sma

	// 计算后续的EMA值
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
	// return ema[period-1:]
}

// Round 四舍五入到指定小数位
func Round(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
