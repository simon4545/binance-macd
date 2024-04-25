package db

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/simon4545/binance-macd/config"
	"github.com/simon4545/binance-macd/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

type Investment struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Currency  string
	Operate   string
	Amount    float64
	Quantity  float64
	UnitPrice float64
}

func InitDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("future.db?_loc=Asia/Shanghai"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&Investment{})
	if err != nil {
		log.Fatal(err)
	}
}

func GetInvestmentCount(currency string) int64 {
	var count int64
	result := db.Model(&Investment{}).Where("currency = ?", currency).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}

func InvestmentAvgPrice(currency string, price, rate float64) bool {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("unit_price as Total").Where("currency = ?", currency).Order("id DESC").Limit(1).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	if rate >= 1 {
		return dbResult.Total == 0 || dbResult.Total/price > rate
	} else {
		return dbResult.Total == 0 || dbResult.Total/price < rate
	}

}

func ClearHistory(currency string) {
	result := db.Exec("DELETE FROM investments where currency = ?", currency)
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
func GetRecentInvestment(currency string, period string) int64 {
	intPeriod := ConvertToSeconds(period)
	current := time.Now().Add(-time.Duration(intPeriod*5) * time.Second)
	var count int64
	result := db.Model(&Investment{}).Where("created_at >= ? and currency = ?", current, currency).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}
func CheckTotalInvestment(conf *config.Config) bool {
	var count int64
	result := db.Model(&Investment{}).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count <= conf.Level
}

type Result struct {
	Total float64
}

func GetSumInvestment(currency string) float64 {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("SUM(amount) as Total").Where("currency = ?", currency).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	// if dbResult.Total == 0 {
	// 	return math.MaxFloat64
	// }
	return dbResult.Total
}
func GetSumInvestmentQuantity(currency string) float64 {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("SUM(quantity) as Total").Where("currency = ?", currency).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return dbResult.Total
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

func MakeDBInvestment(invest Investment) {
	var _invest Investment
	result := db.Where("currency = ?", invest.Currency).Order("id desc").First(&_invest)
	//如果存在记录，则更新
	if result.RowsAffected > 0 && result.Error == nil {
		_invest.Amount = invest.Amount
		_invest.Quantity = invest.Quantity
		_invest.UnitPrice = invest.UnitPrice
		db.Save(&_invest)
	}
}
