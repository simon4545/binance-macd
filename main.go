package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/remeh/sizedwaitgroup"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	go updateATR()
	GetSymbolInfo(client)
	listenWebSocket()
}
func setAtr(symbol string, atr float64, lastClose float64) {
	delock.Lock()
	defer delock.Unlock()
	atrValues[symbol] = atr * cfg.AtrMultier[symbol]
	atrValues[symbol] = math.Max(0.015*lastClose, atrValues[symbol])
	atrPercent := atrValues[symbol] * 100 / lastClose
	log.Printf("%s ATR: %.3f %.2f%%\n", symbol, atrValues[symbol], atrPercent)
}
func fetchATR(symbol string) {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval("2h").Limit(12).Do(context.Background())
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
func hasPumped(symbol string) bool {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval("15m").Limit(60).Do(context.Background())
	if err != nil {
		log.Printf("Error fetching ATR for %s: %v", symbol, err)
		return true
	}
	if len(klines) < 10 {
		return true
	}
	var closes []float64
	for i := 1; i < len(klines); i++ {
		if klines[i].CloseTime > time.Now().UnixMilli() {
			continue
		}
		close := parseFloat(klines[i].Close)
		closes = append(closes, close)
	}

	return closes[len(closes)-1]/closes[0] > 0.03
}
func listenWebSocket() {
	symbols := keys(cfg.Bet)
	symbolsWithInterval := make(map[string]string)
	for _, symbol := range symbols {
		symbolsWithInterval[symbol] = "15m"
	}
	doneC, stopC, err := futures.WsCombinedKlineServe(symbolsWithInterval, func(event *futures.WsKlineEvent) {
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
	if hourCount > 6 {
		return
	}
	db.Create(&Cache{Key: symbol, Value: true})
	if hasPumped(symbol) {
		return
	}
	quantity := roundStepSize(cfg.Bet[symbol]/price, LotSizeMap[symbol])
	order, err := client.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(side)).NewClientOrderID(RandStr("SIM-", 12)).
		Type(futures.OrderTypeMarket).Quantity(fmt.Sprintf("%f", quantity)).NewOrderResponseType(futures.NewOrderRespTypeRESULT).Do(context.Background())
	if err != nil {
		log.Printf("Error placing order: %v", err)
		return
	}
	log.Printf("已开仓: %s %s, %+v\n", symbol, side, order)

	price = parseFloat(order.AvgPrice)
	stopLoss := price * 0.99
	_side := "SELL"
	if side == "SELL" {
		stopLoss = price * 1.01
		_side = "BUY"
	}
	stopLoss = roundStepSize(stopLoss, PriceFilterMap[symbol])
	_, err = client.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(_side)).Type(futures.OrderTypeStopMarket).
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
	_, err = client.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(_side)).Type(futures.OrderTypeTrailingStopMarket).
		NewClientOrderID(RandStr("SIMC-", 12)).ActivationPrice(fmt.Sprintf("%f", takeProfit)).CallbackRate("0.5").Quantity(fmt.Sprintf("%f", quantity)).Do(context.Background())
	if err != nil {
		log.Printf("Error setting take profit: %v", err)
		return
	}
	log.Printf("止盈设置: %f %f %f\n", price, atrValues[symbol], takeProfit)
}
func updateATR() {
	for {
		symbols := keys(cfg.Bet)
		swg := sizedwaitgroup.New(6)
		for _, s := range symbols {
			swg.Add()
			go func(pair string) {
				defer swg.Done()
				fetchATR(pair)
			}(s)
		}
		swg.Wait()
		time.Sleep(5 * time.Minute)
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
		fmt.Println("ws user error:", err)
	}
	var err error
	var doneC chan struct{}
	doneC, wsUserStop, err = futures.WsUserDataServe(listenKey, userWsHandler, errHandler)
	if err != nil {
		fmt.Println("ws user error:", err)
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
		log.Printf("订单回调: %+v\n", message)
		if strings.HasPrefix(message.ClientOrderID, "SIMC-") {
			positonOrder[symbol] = false
			client.NewCancelAllOpenOrdersService().Symbol(symbol).Do(context.Background())
		}
	}
}
