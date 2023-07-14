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
				GoCount: 10000,
				Wait:    true,
			})
			for j := 0; j < 10000; j++ {
				_ = gs.Go(func() error {
					for i := 0; i < 100000; i++ {
						i++
					}
					return nil
				})
			}
		}
	}()
	_ = http.ListenAndServe("localhost:8080", nil)
	fmt.Println("结束")

}
