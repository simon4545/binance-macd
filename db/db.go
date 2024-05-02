package db

import (
	"log"
	"strconv"
	"strings"
	"time"

	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
	"github.com/simon4545/binance-macd/config"
	"github.com/simon4545/binance-macd/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB
var ldb *ledis.DB

type Investment struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	Currency   string
	Operate    string
	Amount     float64
	Quantity   float64
	UnitPrice  float64
	StopLoss   float64
	TakeProfit float64
}
type Log struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Currency  string
	Content   string
}

func InitDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("future.db?_loc=Asia/Shanghai"), &gorm.Config{
		NowFunc: func() time.Time {
			loc, _ := time.LoadLocation("Asia/Shanghai")
			return time.Now().In(loc)
		},
	})

	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&Investment{})
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&Log{})
	if err != nil {
		log.Fatal(err)
	}

	cfg := lediscfg.NewConfigDefault()
	l, _ := ledis.Open(cfg)
	ldb, _ = l.Select(0)
}
func SetInRange(symbol, mode string, inrange bool) {
	if inrange {
		ldb.SetEX([]byte(symbol+mode), 20, []byte{1})
	} else {
		ldb.SetEX([]byte(symbol+mode), 20, []byte{0})
	}
}
func GetInRange(symbol, mode string) (inrange bool) {
	val, err := ldb.Get([]byte(symbol + mode))
	if err != nil || val == nil {
		return
	}
	in := val[0]
	if in == 1 {
		inrange = true
	}
	return
}
func SetOrderCache(symbol string) {
	ldb.SetEX([]byte(symbol), 60*60, []byte("YES"))
}
func GetOrderCache(symbol string) (found bool) {
	val, err := ldb.Get([]byte(symbol))
	if err != nil || val == nil {
		return
	}
	found = true
	return
}
func GetInvestmentCount(currency string, mode string) int64 {
	var count int64
	result := db.Model(&Investment{}).Where("currency = ? and operate= ?", currency, mode).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}
func InvestmentAvgPrice1(currency string, mode string) (dbResult Investment) {
	result := db.Model(&Investment{}).Where("currency = ? and operate= ? ", currency, mode).Order("id DESC").Limit(1).Find(&dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	return
}
func InvestmentAvgPrice(currency string, price, rate float64, mode string) bool {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("unit_price as Total").Where("currency = ? and operate= ? ", currency, mode).Order("id DESC").Limit(1).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	if strings.HasSuffix(mode, "LONG") {
		return dbResult.Total == 0 || price <= (dbResult.Total-rate*1.5)
	} else {
		return dbResult.Total == 0 || price >= (dbResult.Total+rate*1.5)
	}
}

func ClearHistory(currency, mode string) {
	var result *gorm.DB
	if mode != "" {
		result = db.Exec("DELETE FROM investments where currency = ? and operate = ? ", currency, mode)
	} else {
		result = db.Exec("DELETE FROM investments where currency = ? ", currency)
	}

	if result.Error != nil {
		log.Fatal(result.Error)
	}
}
func ConvertToSeconds(s string) int {
	sValue := strings.Split(s, "")
	sNewStr := sValue[:len(sValue)-1]
	i, err := strconv.Atoi(strings.Join(sNewStr, ""))
	if err != nil {
		panic(err)
	}
	return i * utils.SecondsPerUnit[sValue[len(sValue)-1]]
}
func GetRecentInvestment(currency string, period string, mode string) int64 {
	intPeriod := ConvertToSeconds(period)
	current := time.Now().Add(-time.Duration(intPeriod*5) * time.Second)
	var count int64
	result := db.Model(&Investment{}).Where("created_at >= ? and currency = ? and operate= ? ", current, currency, mode).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}
func CheckTotalInvestment(conf *config.Config, mode string) bool {
	var count int64
	result := db.Model(&Investment{}).Where("operate= ? ", mode).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count <= conf.Level
}

type Result struct {
	Total float64
}

func GetSumInvestment(currency string, mode string) float64 {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("SUM(amount) as Total").Where("currency = ? and operate= ? ", currency, mode).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	// if dbResult.Total == 0 {
	// 	return math.MaxFloat64
	// }
	return dbResult.Total
}
func GetSumInvestmentQuantity(currency string, mode string) float64 {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("SUM(quantity) as Total").Where("currency = ? and operate= ?", currency, mode).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return dbResult.Total
}
func Insert(currency string, amount float64, quantity, price, tp, sl float64, side string) {
	investment := Investment{
		Operate:    side,
		Currency:   currency,
		Amount:     amount,
		Quantity:   quantity,
		UnitPrice:  price,
		TakeProfit: tp,
		StopLoss:   sl,
	}
	result := db.Create(&investment)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	SetOrderCache(currency)
}
func InsertInvestment(currency string, amount float64, quantity, price float64, side string) {
	investment := Investment{
		Operate:   side,
		Currency:  currency,
		Amount:    amount,
		Quantity:  quantity,
		UnitPrice: price,
	}
	result := db.Create(&investment)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func MakeLog(symbol, content string) {
	log := &Log{}
	log.Content = content
	log.Currency = symbol
	db.Save(log)
}
