package main

import (
	"log"
	"strconv"
	"strings"
	"time"

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
	db, err = gorm.Open(sqlite.Open("investment.db?_loc=Asia/Shanghai"), &gorm.Config{})
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

func InvestmentAvgPrice(currency string, price float64) bool {
	dbResult := &Result{}
	result := db.Model(&Investment{}).Select("(amount/quantity) as Total").Where("currency = ?", currency).Order("id DESC").Limit(1).Scan(dbResult)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return dbResult.Total == 0 || dbResult.Total/price > 1.05
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
	return i * SecondsPerUnit[sValue[len(sValue)-1]]
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
func CheckTotalInvestment() bool {
	var count int64
	result := db.Model(&Investment{}).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count < 10
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
func InsertInvestment(currency string, amount float64, quantity float64) {
	investment := Investment{
		Operate:   "BUY",
		Currency:  currency,
		Amount:    amount,
		Quantity:  quantity,
		UnitPrice: amount / quantity,
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
