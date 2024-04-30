package interfacer

import (
	"github.com/adshao/go-binance/v2/futures"
	"github.com/simon4545/binance-macd/config"
)

type Executor interface {
	Handle(*futures.Client, *config.Config, string, float64, []float64, []float64, []float64, []float64)
	CreateSellSide(*futures.Client, *config.Config, string, float64)
	CreateBuySide(*futures.Client, *config.Config, string, float64, float64)
}

var factoryByName = make(map[string]Executor)

func Register(name string, factory Executor) {
	factoryByName[name] = factory
}

func Create(name string, client *futures.Client) Executor {
	if name == "BOTH" {
		panic("BOTH模式不再支持，请改用LONG|SHORT写法，中间是|符号")
	}
	if f, ok := factoryByName[name]; ok {
		return f
	} else {
		panic("无效的模式，目前仅支持 LONG SHORT FASTSHORT FASTLONG ，可以同时使用，用|隔开")
	}
}
