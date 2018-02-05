package main

import (
	"math/rand"
        "time"
)

func SellSellSell() {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating SellSellSell order")
		return
	}
}

func Buy(base, coin string, price, basebalance float64) {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating buy order")
		if rand.Intn(20) == 0 {
			Warning.Print("Simulated Buy failed (random chance in 20)")
			return
		}
		// assume a buy completes (to make simulation work!)
		if state[base].Balance<basebalance { 
                    Warning.Print("Logic error - base balance is too low to actually purchase a coin!")
                    return 
                }
		state["LAST"].Coin = coin
		state[coin].PurchasePrice = price
		coinbalance := basebalance*(1 - conf.GetFloat64("TradingRules.buyfee"))/price
		

		state[coin].Balance += coinbalance
		state[coin].Date = time.Now()
		// TODO update date
		state[base].Balance -= basebalance
		// TODO update date
		if state[base].Balance < 0 {
			state[base].Balance = 0
		}
		Info.Printf("Order placed for %v of %v at %v (paid %v %v)\n", fc(coinbalance), coin, fc(price), fc(basebalance), base)
		return
	}
}

func Sell(base, coin string, price, coinbalance float64) {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating Sell order")

		if rand.Intn(20) == 0 {
			Warning.Print("Simulated Sell failed (random chance in 20)")
			return
		}
		// assume a sale completes (to make simulation work!)
		state["LAST"].Coin = base
		state[coin].PurchasePrice = price
		//value:=price*coinbalance
		valueafterfees := price * (1 - conf.GetFloat64("TradingRules.sellfee")) * coinbalance

		state[coin].Balance -= coinbalance
		// TODO update date
		state[base].Balance += valueafterfees
		// TODO update date
		if state[coin].Balance < 0 {
			state[coin].Balance = 0
		}
		Info.Printf("Sell Order placed for %v of %v at %v (received %v %v)\n", fc(coinbalance), coin, fc(price), fc(valueafterfees), base)
		return
	}
}

func CancelOrder() {
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulating Cancel order")
		return
	}
}
