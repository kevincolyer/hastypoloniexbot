package main

import "time"

type tick struct{}

var throttlerchan chan tick

func throttler(t chan tick) {
	for true {
		_ = <-t
		time.Sleep(time.Second / 6)
	}
	return
}

func throttle() {
	throttlerchan <- tick{}
	return
}
