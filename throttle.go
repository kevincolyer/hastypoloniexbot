package main

import "time"

// global throttle to limit api calls to 6 per second

type tick struct{}

var throttlerchan chan tick 


func throttler(t chan tick) {
	for true {
		_ = <-t
		time.Sleep(time.Second / 6)
	}
	return
}

func Throttle() {
	throttlerchan <- tick{}
	return
}
