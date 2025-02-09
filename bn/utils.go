package bn

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/simon4545/binance-macd/configuration"
)

func GetBalance(client *binance.Client, token string) float64 {
	res, err := client.NewGetAccountService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1
	}
	balance := 0.0
	// fmt.Println(res.Balances)
	for _, s := range res.Balances {
		if s.Asset == token {
			balance, _ = strconv.ParseFloat(s.Free, 64)
		}
	}
	return balance
}

func GetSymbolInfo(client *binance.Client) {
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		print("Error fetching exchange info:", err)
		os.Exit(1)
	}
	for _, s := range info.Symbols {
		// if strings.HasSuffix(s.Symbol, "USDT") {
		lotSizeFilter := s.LotSizeFilter()
		quantityTickSize, _ := strconv.ParseFloat(lotSizeFilter.StepSize, 64)
		configuration.LotSizeMap[s.Symbol] = quantityTickSize
		priceFilter := s.PriceFilter()
		priceTickSize, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
		configuration.PriceFilterMap[s.Symbol] = priceTickSize
		// return
		// }
	}

	rate, err := client.NewTradeFeeService().Do(context.Background())
	if err != nil {
		print("Error fetching trade fee:", err)
		os.Exit(1)
	}
	for _, s := range rate {
		fee, _ := strconv.ParseFloat(s.TakerCommission, 64)
		configuration.FeeMap[s.Symbol] = fee
	}
}
