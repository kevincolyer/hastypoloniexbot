package main

import (
	"math/rand"
	"strconv"
	"time"
)

// func SellSellSell() {
// 	if b.Conf.GetBool("BotControl.Simulate") {
// 		if b.Logging { Info.Println("Simulating SellSellSell order") }
// 		return
// 	}
// 	//	throttle()
// 	if b.Logging { Warning.Println("SellSellSell Not Implemented yet") }
// 	return
// }

func (b *Bot) Buy(base, coin string, price, basebalance float64) {
	coinbalance := basebalance * (1 - b.Conf.GetFloat64("TradingRules.buyfee")) / price
	if b.Conf.GetBool("BotControl.Simulate") {
		// 		if b.Logging { Info.Println("Simulating buy order") }
		if rand.Intn(20) == 0 {
			b.LogWarning(coin + " Simulated Buy failed (random chance in 20)")
			return
		}
		// assume a buy completes (to make simulation work!)
		if b.State[base].Balance < basebalance {
			b.LogWarning("Logic error - base balance is too low to actually purchase a coin!")
			return
		}
		b.State[LAST].Coin = coin
		b.State[coin].PurchasePrice = price

		b.State[coin].Balance += coinbalance
		b.State[coin].Coin = coin
		b.State[coin].Date = time.Now()
		// TODO update date
		b.State[base].Balance -= basebalance
		// TODO update date
		if b.State[base].Balance < 0 {
			b.State[base].Balance = 0
		}
		b.LogInfof(coin+" Buy  order placed for %v of %v at %v (paid %v %v)", fc(coinbalance), coin, fc(price), fc(basebalance), base)
		return
	}
	////////////////////////////////////////////////
	// Actual order

	//Buy(pair string, rate, amount float64) (buy Buy, err error) {
	b.LogInfof(coin+" Buy  order placed for %v of %v at %v (paid %v %v probably) ", fc(coinbalance), coin, fc(price), fc(basebalance), base)
	Throttle()
	buyorder, err := b.Exchange.Buy(base+"_"+coin, price, basebalance/price)
	// placing this below so coins that follow don't use all the balance...
	b.State[base].Balance -= basebalance
	if err != nil {
		b.LogWarningf(coin+" BUY  order failed for %v with error: %v", coin, err)
		return
	}
	if buyorder.OrderNumber == 0 {
		b.LogWarning(coin + " BUY  order was not placed at exchange")
		return
	}
	b.State[coin].Date = time.Now()
	b.State[coin].PurchasePrice = price
	b.State[coin].OrderNumber = strconv.FormatInt(buyorder.OrderNumber, 10)
	b.State[coin].Balance += basebalance / price
	return
}

func (b *Bot) Sell(base, coin string, price, coinbalance float64) {
	valueafterfees := price * (1 - b.Conf.GetFloat64("TradingRules.sellfee")) * coinbalance
	if b.Conf.GetBool("BotControl.Simulate") {
		//if b.Logging { Info.Println("Simulating Sell order") }

		if rand.Intn(20) == 0 {
			b.LogWarning(coin + " Simulated Sell failed (random chance in 20)")
			return
		}
		// assume a sale completes (to make simulation work!)
		b.State[LAST].Coin = base
		b.State[coin].PurchasePrice = price
		b.State[coin].SaleDate = time.Now()
		//value:=price*coinbalance

		b.State[coin].Balance -= coinbalance
		// TODO update date
		b.State[base].Balance += valueafterfees
		// TODO update date
		if b.State[coin].Balance < 0 {
			b.State[coin].Balance = 0
		}

		b.LogInfof(coin+" Sell order placed for %v of %v at %v (received %v %v)", fc(coinbalance), coin, fc(price), fc(valueafterfees), base)
		return
	}
	////////////////////////////////////////////////
	// Actual order

	//Buy(pair string, rate, amount float64) (buy Buy, err error) {
	b.LogInfof(coin+" SELL order placed for %v of %v at %v (recieved %v %v probably)", fc(coinbalance), coin, fc(price), fc(valueafterfees), base)
	Throttle()
	sellorder, err := b.Exchange.Sell(base+"_"+coin, price, coinbalance)
	if err != nil {
		b.LogWarningf(coin+" SELL order failed for %v with error: %v", coin, err)
		return
	}

	if sellorder.OrderNumber == 0 {
		b.LogWarningf(coin + " SELL order was not placed at exchange")
		return
	}
	// provisional values - sale might not go ahead!
	b.State[coin].Balance = 0
	b.State[base].Balance += valueafterfees
	b.State[coin].SaleDate = time.Now()
}

func (b *Bot) CancelAllOpenOrders(base string, targets []string) (ok bool) {
	ok = true
	if b.Conf.GetBool("BotControl.simulate") {
		b.LogInfo("CANCEL ALL ORDERS - Simulated ok")
		return
	}
	for _, coin := range targets {
		Throttle()
		oos, err := b.Exchange.OpenOrders(base + "_" + coin)
		if err != nil {
			ok = false
			b.LogWarningf("Getting OpenOrders failed with error: %v", err)
		}

		for _, o := range oos {
			success, err := b.Exchange.CancelOrder(o.OrderNumber)
			if success == false {
				ok = false
				b.LogWarningf("CancelOrder failed with error: %v", err)
			}
		}
	}
	return
}
