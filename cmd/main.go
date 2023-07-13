package main

import (
	"fmt"
	threading "go-threading"
	"net/http"
	_ "net/http/pprof"
	"time"
)

func main() {
	threading.StartPool(threading.SetMaxIdleWorkerDuration(time.Second*30), threading.SetMinWorkCount(10000))
	go func() {
		for {
			gs := threading.New(threading.Config{
				GoCount: 1000,
				Wait:    true,
			})
			for j := 0; j < 100; j++ {
				_ = gs.Go(func() error {
					time.Sleep(time.Microsecond * 100)
					return nil
				})
			}
		}
	}()
	_ = http.ListenAndServe("localhost:8080", nil)
	fmt.Println("结束")

}
