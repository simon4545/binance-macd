package bn

import (
	"fmt"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

var priceUpdateLocker sync.Mutex
var client *futures.Client
var Amplitudes = make(map[string]float64)
var Atrs = make(map[string]float64)
var SymbolPrice = make(map[string][]float64)
var Symbols = []string{"BTCUSDT", "XRPUSDT", "SOLUSDT", "DOGEUSDT"}

func Init(fclient *futures.Client) {
	client = fclient
	InitWS(client)
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for t := range ticker.C {
			HandleUpdatePrice()
			fmt.Println("定时任务执行，当前时间：", t)
			// for _, k := range Symbols {
			// 	assetInfo := AssetInfo[k]
			// 	Handle(k, assetInfo)
			// }
		}
	}()
	// go functions.CheckCross(client, config.Symbols, config, Handle)
}
func HandleUpdatePrice() {
	priceUpdateLocker.Lock()
	defer priceUpdateLocker.Unlock()
	for _, k := range Symbols {
		lenN := len(SymbolPrice[k])
		if lenN > 600 {
			SymbolPrice[k] = SymbolPrice[k][lenN-500:]
		}
	}
}
