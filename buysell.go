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
		value:=price*amount
		amountafterfees:=value*(1-conf.GetFloat64("TradingRules.buyfee"))/price
		
		state[coin].Balance += amountafterfees
		// TODO update date
		state[base].Balance -= value
		// TODO update date
		if state[base].Balance < 0 {
			state[base].Balance = 0
		}
		Info.Printf("Order placed for %v of %v at %v (paid %v %v)\n", fn(amountafterfees), coin, fc(price), fc(value), base)
		return
	}
}

func Sell(base, coin string, price, amount float64) {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating Sell order")

		if rand.Intn(20) == 0 {
			Warning.Print("Simulated Sell failed (random chance in 20)")
			return
		}
		// assume a sale completes (to make simulation work!)
		state["LAST"].Coin = base
		state[coin].PurchasePrice = price
		//value:=price*amount
		valueafterfees:=price*(1-conf.GetFloat64("TradingRules.sellfee"))*amount
		
		state[coin].Balance -= amount
		// TODO update date
		state[base].Balance -= valueafterfees
		// TODO update date
		if state[coin].Balance < 0 {
			state[coin].Balance = 0
		}
		Info.Printf("Sell Order placed for %v of %v at %v (paid %v %v)\n", fn(amount), coin, fc(price), fc(valueafterfees), base)
		return
	}
}

func CancelOrder() {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating Cancel order")
		return
	}
}
