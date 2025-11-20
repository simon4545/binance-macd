package db

import (
	"log"
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

type Earn struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	DateForm  string `gorm:"uniqueIndex:idx_currency_date"`
	Currency  string `gorm:"uniqueIndex:idx_currency_date"`
	Count     int    `gorm:"default:0"`
	Amount    float64
}
type DBAmount struct {
	Currency string
	Amount   float64
}

func InitDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("investment.db?_loc=Asia/Shanghai"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&Investment{}, &Earn{})
	if err != nil {
		log.Fatal(err)
	}
	if errv := db.Exec(`
		CREATE VIEW IF NOT EXISTS investment_summary AS
		SELECT 
			currency,
			SUM(amount) AS total_amount,
			SUM(quantity) AS total_quantity,
			CASE 
				WHEN SUM(quantity) = 0 THEN 0
				ELSE SUM(amount) / SUM(quantity)
			END AS weighted_unit_price
		FROM 
			investments
		GROUP BY currency;`).Error; errv != nil {
		log.Fatal(err)
	}
}
func GetAllInvestments() (invests []Investment) {
	result := db.Order("id desc").Limit(20).Find(&invests)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return
}
func GetAllEarns() (earns []Earn) {
	result := db.Order("id desc").Limit(50).Find(&earns)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return
}
func GetInvestments(currency string) (invests []Investment) {
	result := db.Model(&Investment{}).Where("currency = ?", currency).Find(&invests)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return
}
func GetTotalAmount() (totalAmount []DBAmount) {

	query := db.Model(&Investment{}).Select("Currency,SUM(amount)  as Amount")
	result := query.Group("currency").Scan(&totalAmount)
	if result.Error != nil {
		log.Fatal("Query failed:", result.Error)
	}
	return
}
func GetTotalEarn() (totalAmount []DBAmount) {
	query := db.Model(&Earn{}).Select("Currency,SUM(amount) as Amount")
	result := query.Group("currency").Scan(&totalAmount)
	if result.Error != nil {
		log.Fatal("Query failed:", result.Error)
	}
	return
}
func ClearHistory(currency string, earn float64) {
	result := db.Exec("DELETE FROM investments where currency = ?", currency)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	currentTime := time.Now()

	formattedDate := currentTime.Format("2006-01-02")
	// 插入或更新 earn 表
	result = db.Exec(`
		INSERT INTO earns (currency, date_form, amount,count) VALUES (?,?,?,1)
		ON CONFLICT(currency,date_form) DO UPDATE SET amount = amount + EXCLUDED.amount, count = count + 1
	`, currency, formattedDate, earn)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func ClearHistoryById(id uint, currency string, earn float64) {
	result := db.Exec("DELETE FROM investments where id = ?", id)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	currentTime := time.Now()
	formattedDate := currentTime.Format("2006-01-02")
	result = db.Exec(`
		INSERT INTO earns (currency,date_form, amount,count) VALUES (?,?,?,1)
		ON CONFLICT(currency,date_form) DO UPDATE SET amount = amount + EXCLUDED.amount, count = count + 1
	`, currency, formattedDate, earn)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func InsertInvestment(currency string, amount float64, quantity, price float64) {
	investment := Investment{
		Operate:   "BUY",
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
