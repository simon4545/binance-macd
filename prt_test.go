package main

import (
	"fmt"
	"testing"

	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
)

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
