package main

import (
	"math/rand"
)

func SellSellSell() {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating SellSellSell order")
		return
	}
}

func Buy(base, coin string, price, amount float64) {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating buy order")
		if rand.Intn(20) == 0 {
			Warning.Print("Simulated Buy failed (random chance in 20)")
			return
		}
		// assume a buy completes (to make simulation work!)
		state["LAST"].Coin = coin
		state[coin].PurchasePrice = price
		state[coin].Balance += amount
		// TODO update date
		state[base].Balance -= price * amount
		// TODO update date
		if state[base].Balance < 0 {
			state[base].Balance = 0
		}
		Info.Printf("Order placed for %v of %v at %v (paid %v %v)\n", amount, coin, price, amount*price, base)
		return
	}
}

func Sell() {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating Sell order")

		if rand.Intn(20) == 0 {
			Warning.Print("Simulated Sell failed (random chance in 20)")
			return
		}
		return
	}
}

func CancelOrder() {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating Cancel order")
		return
	}
}
