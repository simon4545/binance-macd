package main

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

var config *Config

type Config struct {
	BAPI_KEY   string   `yaml:"BAPI_KEY"`
	BAPI_SCRET string   `yaml:"BAPI_SCRET"`
	Symbols    []string `yaml:"SYMBOLS"`
	Amount     float64  `yaml:"AMOUNT"`
}

func readConfig(c *Config) {
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
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
