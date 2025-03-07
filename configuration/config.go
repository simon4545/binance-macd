package configuration

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

var SecondsPerUnit map[string]int = map[string]int{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}
var LotSizeMap = map[string]float64{}
var PriceFilterMap = map[string]float64{}
var FeeMap = map[string]float64{}
var AtrMap = map[string]float64{}

type SymbolConfig struct {
	Amount string  `yaml:"AMOUNT"`
	Multi  float64 `yaml:"MULTI"`
	Period string  `yaml:"PERIOD"`
}
type Config struct {
	BAPI_KEY   string                   `yaml:"BAPI_KEY"`
	BAPI_SCRET string                   `yaml:"BAPI_SCRET"`
	Symbols    map[string]*SymbolConfig `yaml:"SYMBOLMAP"`
}

func isPortAvailable(port int) bool {
	// 尝试在指定端口上监听
	address := ":" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		// 如果出现错误，表示端口被占用
		return false
	}
	// 关闭监听器
	defer listener.Close()
	return true
}

func findAvailablePort(startPort int) int {
	port := startPort
	for {
		if isPortAvailable(port) {
			return port
		}
		port++
	}
}

func (c *Config) Read() {
	yamlFile, err := os.ReadFile("future.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	// if c.Port == 0 {
	// 	c.Port = findAvailablePort(8888)
	// }
	for _, v := range c.Symbols {
		if v.Multi == 0 {
			v.Multi = 2.5
		}
		if v.Period == "" {
			v.Period = "4h"
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
