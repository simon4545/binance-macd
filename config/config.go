package config

import (
	"log"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

var OrderLocker sync.Mutex
var AtrMap map[string]float64
var LotSizeMap map[string]float64
var PriceFilterMap map[string]float64
var FeeMap map[string]float64

type Config struct {
	BAPI_KEY   string   `yaml:"BAPI_KEY"`
	BAPI_SCRET string   `yaml:"BAPI_SCRET"`
	Symbols    []string `yaml:"SYMBOLS"`
	Amount     float64  `yaml:"AMOUNT"`
	Exclude    []string `yaml:"EXCLUDE"`
	Period     string   `yaml:"PERIOD"`
	Level      int64    `yaml:"LEVEL"`
	Side       string   `yaml:"SIDE"`
}

func readConfig(c *Config) {
	yamlFile, err := os.ReadFile("future.yaml")
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
		c.Period = "1m"
	}
	if c.Side == "" {
		c.Side = "Long"
	}
}
func InitConfig(c *Config) {
	readConfig(c)
	go func() {
		for {
			readConfig(c)
			time.Sleep(time.Second * 60)
		}
	}()
}
