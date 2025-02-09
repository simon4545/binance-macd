package configuration

import (
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

type Config struct {
	BAPI_KEY     string   `yaml:"BAPI_KEY"`
	BAPI_SCRET   string   `yaml:"BAPI_SCRET"`
	Symbols      []string `yaml:"SYMBOLS"`
	Amount       float64  `yaml:"AMOUNT"`
	Multi        float64  `yaml:"MULTI"`
	Exclude      []string `yaml:"EXCLUDE"`
	Period       string   `yaml:"PERIOD"`
	Level        int      `yaml:"LEVEL"`
	PriceProtect float64  `yaml:"PRICEPROTECT"`
	ForceSell    float64  `yaml:"FORCESELL"`
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
	if c.Level == 0 {
		c.Level = 20
	}
	if c.Period == "" {
		c.Period = "4h"
	}
	if c.PriceProtect == 0.0 {
		c.PriceProtect = 0.015
	}
	if c.ForceSell == 0.0 {
		c.ForceSell = 0.1
	}
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
