package main

import (
	"math/rand"
	"strconv"
	"time"
)

// func SellSellSell() {
// 	if conf.GetBool("BotControl.Simulate") {
// 		Info.Println("Simulating SellSellSell order")
// 		return
// 	}
// 	//	throttle()
// 	Warning.Println("SellSellSell Not Implemented yet")
// 	return
// }

func Buy(base, coin string, price, basebalance float64) {
	coinbalance := basebalance * (1 - conf.GetFloat64("TradingRules.buyfee")) / price
	if conf.GetBool("BotControl.Simulate") {
		// 		Info.Println("Simulating buy order")
		if rand.Intn(20) == 0 {
			Warning.Print(coin + " Simulated Buy failed (random chance in 20)")
			return
		}
		// assume a buy completes (to make simulation work!)
		if state[base].Balance < basebalance {
			Warning.Print("Logic error - base balance is too low to actually purchase a coin!")
			return
		}
		state[LAST].Coin = coin
		state[coin].PurchasePrice = price

		state[coin].Balance += coinbalance
		state[coin].Coin = coin
		state[coin].Date = time.Now()
		// TODO update date
		state[base].Balance -= basebalance
		// TODO update date
		if state[base].Balance < 0 {
			state[base].Balance = 0
		}
		Info.Printf(coin+" Buy  order placed for %v of %v at %v (paid %v %v)\n", fc(coinbalance), coin, fc(price), fc(basebalance), base)
		return
	}
	////////////////////////////////////////////////
	// Actual order

	//Buy(pair string, rate, amount float64) (buy Buy, err error) {
	Info.Printf(coin+" Buy  order placed for %v of %v at %v (paid %v %v probably)\n", fc(coinbalance), coin, fc(price), fc(basebalance), base)
	throttle()
	buyorder, err := exchange.BuyPostOnly(base+"_"+coin, price, basebalance/price)
	if err != nil {
		Warning.Printf(coin+" BUY  order failed for %v with error: %v\n", coin, err)
		return
	}
	if buyorder.OrderNumber==0 {
		Warning.Printf(coin+ " BUY  order was not placed at exchange\n")
		return
        }
	state[coin].Date = time.Now()
	state[coin].PurchasePrice = price
	state[coin].OrderNumber = strconv.FormatInt(buyorder.OrderNumber, 10)
	state[base].Balance -= basebalance

	return
}

func Sell(base, coin string, price, coinbalance float64) {
	valueafterfees := price * (1 - conf.GetFloat64("TradingRules.sellfee")) * coinbalance
	if conf.GetBool("BotControl.Simulate") {
		//Info.Println("Simulating Sell order")

		if rand.Intn(20) == 0 {
			Warning.Print(coin + " Simulated Sell failed (random chance in 20)")
			return
		}
		// assume a sale completes (to make simulation work!)
		state[LAST].Coin = base
		state[coin].PurchasePrice = price
		//value:=price*coinbalance

		state[coin].Balance -= coinbalance
		// TODO update date
		state[base].Balance += valueafterfees
		// TODO update date
		if state[coin].Balance < 0 {
			state[coin].Balance = 0
		}

		Info.Printf(coin+" Sell order placed for %v of %v at %v (received %v %v)\n", fc(coinbalance), coin, fc(price), fc(valueafterfees), base)
		return
	}
	////////////////////////////////////////////////
	// Actual order

	//Buy(pair string, rate, amount float64) (buy Buy, err error) {
	Info.Printf(coin+" SELL order placed for %v of %v at %v (recieved %v %v probably)\n", fc(coinbalance), coin, fc(price), fc(valueafterfees), base)
	throttle()
	sellorder, err := exchange.Sell(base+"_"+coin, price, coinbalance)
	if err != nil {
		Warning.Printf(coin+" SELL order failed for %v with error: %v\n", coin, err)
		return
	}
	
	if sellorder.OrderNumber==0 {
		Warning.Printf(coin+ " SELL order was not placed at exchange\n")
		return
        }
	// provisional values - sale might not go ahead!
	state[coin].Balance = 0
	state[base].Balance += valueafterfees

}

func CancelAllOpenOrders(base string, targets []string) (ok bool) {
	ok = true
	if conf.GetBool("BotControl.simulate") {
		Info.Println("CANCEL ALL ORDERS - Simulated ok")
		return
	}
	for _, coin := range targets {
		throttle()
		oos, err := exchange.OpenOrders(base + "_" + coin)
		if err != nil {
			ok = false
			Warning.Printf("Getting OpenOrders failed with error: %v\n", err)
		}

		for _, o := range oos {
			success, err := exchange.CancelOrder(o.OrderNumber)
			if success == false {
				ok = false
				Warning.Printf("CancelOrder failed with error: %v\n", err)
			}
		}
	}
	return
}
