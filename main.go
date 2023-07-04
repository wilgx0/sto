package main

import (
	"fmt"
	"github.com/wilgx0/sto/sto"
	"time"
)

func main() {
	now := time.Now()

	s := sto.New()
	s.Start()
	id6 := s.AddFunc(now.Add(time.Second*6), func(payload interface{}) {
		fmt.Println(payload)
	}, 6)

	_ = id6
	id4 := s.AddFunc(now.Add(time.Second*4), func(payload interface{}) {
		fmt.Println(payload)
	}, 4)

	_ = id4

	s.AddFunc(now.Add(time.Second*1), func(payload interface{}) {
		fmt.Println(payload)
		s.Remove(id6)
		s.UpdateFunc(id4, now.Add(time.Second*6), func(payload interface{}) {
			fmt.Println(payload)
		}, 4)

		s.AddFunc(now.Add(time.Second*5), func(payload interface{}) {
			fmt.Println(payload)
		}, 5)

		s.AddFunc(now.Add(time.Second*7), func(payload interface{}) {
			fmt.Println(payload)
		}, 7)
	}, 1)

	s.AddFunc(now.Add(time.Second*2), func(payload interface{}) {
		fmt.Println(payload)
	}, 2)

	select {}
}
