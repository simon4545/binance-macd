package configuration

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

var SecondsPerUnit map[string]int = map[string]int{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}
var LotSizeMap map[string]float64
var PriceFilterMap map[string]float64
var FeeMap map[string]float64
var AtrMap map[string]float64

type SymbolConfig struct {
	Amount       float64 `yaml:"AMOUNT"`
	Multi        float64 `yaml:"MULTI"`
	Period       string  `yaml:"PERIOD"`
	Level        int     `yaml:"LEVEL"`
	PriceProtect float64 `yaml:"PRICEPROTECT"`
	ForceSell    float64 `yaml:"FORCESELL"`
}
type Config struct {
	BAPI_KEY   string                   `yaml:"BAPI_KEY"`
	BAPI_SCRET string                   `yaml:"BAPI_SCRET"`
	Symbols    map[string]*SymbolConfig `yaml:"SYMBOLMAP"`
}

func (c *Config) Read() {
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	for _, v := range c.Symbols {
		if v.Amount == 0.0 {
			v.Amount = 100
		}
		if v.Multi == 0.0 {
			v.Multi = 0.1
		}
		if v.Level == 0 {
			v.Level = 20
		}
		if v.Period == "" {
			v.Period = "5m"
		}
		if v.PriceProtect == 0.0 {
			v.PriceProtect = 0.015
		}
		if v.ForceSell == 0.0 {
			v.ForceSell = 0.1
		}
	}
	fmt.Println(c)
}
func (c *Config) Init() {
	c.Read()
	go func() {
		for {
			c.Read()
			time.Sleep(time.Second * 60)
		}
	}()
}
