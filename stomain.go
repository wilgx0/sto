package main

import (
	"Sto/sto"
	"fmt"
	"time"
)

func main() {
	cron := sto.New()
	cron.AddFunc(time.Now().Add(time.Second*3), func(payload interface{}) {
		fmt.Println(payload)
	}, "test")
	cron.Start()
	select {}
}