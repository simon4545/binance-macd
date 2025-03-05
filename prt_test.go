package main

import (
	"fmt"
	"math/big"
	"testing"

	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
)

func IntToBytes(i int) []byte {
	if i > 0 {
		return append(big.NewInt(int64(i)).Bytes(), byte(1))
	}
	return append(big.NewInt(int64(i)).Bytes(), byte(0))
}
func BytesToInt(b []byte) int {
	if b == nil {
		return 0
	}
	if b[len(b)-1] == 0 {
		return -int(big.NewInt(0).SetBytes(b[:len(b)-1]).Int64())
	}
	return int(big.NewInt(0).SetBytes(b[:len(b)-1]).Int64())
}

func TestConvert(t *testing.T) {
	result := IntToBytes(0)
	fmt.Println(result)
	fmt.Println(BytesToInt(result))

}
func TestPrc(t *testing.T) {
	cfg := lediscfg.NewConfigDefault()
	l, _ := ledis.Open(cfg)
	db, _ := l.Select(0)
	db.Incr([]byte("test"))
	result, _ := db.Get([]byte("test"))
	if result != nil {
		fmt.Println(string(result))

		db.Incr([]byte("test"))
	}

}
