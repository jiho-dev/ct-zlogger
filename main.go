package main

import (
	"fmt"

	"github.com/zlogger/pkg/zlog"
)

func main() {
	var path string = "/proc/zlog"

	fmt.Printf("path: %s \n", path)

	memOrder := 16 // 65KB
	logger, err := zlog.NewZlogger(path, memOrder)
	if err != nil {
		fmt.Printf("cannot open Zlogger, err: %s \n", err)
		return
	}

	defer logger.Close()

	zread, err := logger.ReadLog()
	if err != nil {
		fmt.Printf("cannot read log from kernel, err: %s \n", err)
		return
	}

	fmt.Printf("zread.Owner %d\n", zread.Owner)
	fmt.Printf("zread.Start: %d \n", zread.Start)
	fmt.Printf("zread.Count: %d \n", zread.Count)

	var i uint16
	for i = 0; i < zread.Count; i++ {
		idx := i + zread.Start
		zlog := logger.GetLog(idx)

		fmt.Printf("zlog[%d]: Owner:%d, %s \n", i, zlog.Owner, zlog.GetMessage())
	}

	logger.DoneReadLog(zread.Start + zread.Count)
}
