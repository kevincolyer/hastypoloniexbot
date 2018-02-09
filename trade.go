package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gitlab.com/wmlph/poloniex-api"
)

func trade() {
	/////////////////////////////////////
	// get poloniex data and set up variables from config file
	Info.Println("GETTING POLONIEX DATA")

	//Info.Printf("%v\n%v\n", conf.GetString("Credentials.apikey"),conf.GetString("Credentials.secret"))
	exchange = poloniex.NewKS(
		conf.GetString("Credentials.apikey"),
		conf.GetString("Credentials.secret")) // check for failure needed here?

	// Get Ticker
	// {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
	ticker, err := exchange.Ticker()
	if err != nil {
		Error.Printf("Fatal error getting ticker data from poloniex: %v\n", err)
		return
	}

	fiat := conf.GetString("Currency.Fiat")
	FIATBTC := ticker[fiat+"_BTC"].Last // can be other curency than usdt
	base := conf.GetString("Currency.Base")
	sss := conf.GetBool("BotControl.SellSellSell")
	simulate := conf.GetBool("BotControl.Simulate")
	var coinbalance float64
	var fragmenttotal float64 // how many coins we will split our base coin into
	var coin string
	var todo []coinaction
	var balances poloniex.AvailableAccountBalances

	//////////////////////////////////////
	// get current values and prices

	if simulate == false {
		// 		balances, err = exchange.BalancesAll()
		balances, err = exchange.AvailableAccountBalances()
		if err != nil {
			Error.Printf("Failed to get coin AccountBalances from poloniex: %v\n", err)
		}
		state[base].Balance = balances.Exchange[base] // + balances[base].OnOrders // include open buy orders - they will get cancelled below. Posssible race condition here!
	}

	basebalance := state[base].Balance // SIMULATION
	basetotal := basebalance
	state[base].FiatValue = basebalance * FIATBTC
	state[base].BaseValue = basebalance // for completeness

	// get list of coins we are targetting
	targets := getTargettedCoins()

	for _, coin = range targets {
		// if we have not loaded this coin from json, we need to add it to the map.
		if _, ok := state[coin]; !ok {
			state[coin] = &coinstate{Coin: coin, Balance: 0.0}
		}

		if simulate {
			coinbalance = state[coin].Balance // SIMULATION!
		} else {
			// 			coinbalance = balances[coin].Available + balances[coin].OnOrders // REAL THING!
			coinbalance = balances.Exchange[coin] // REAL THING!
		}

		infiat := ticker[base+"_"+coin].Last * coinbalance * FIATBTC
		inbase := ticker[base+"_"+coin].Last * coinbalance
		// 		Info.Printf("BALANCE %v %v (%v %v) \n", fc(coinbalance), coin, fc(infiat), fiat)

		action := NOACTION
		if coinbalance > 0 {
			fragmenttotal++
			basetotal += inbase
			if sss == true {
				action = SELL
			}
		}
		todo = append(todo, coinaction{coin: coin, action: action}) // used to prepare todo slice for use in sellsellsell
		state[coin].Balance = coinbalance
		state[coin].FiatValue = infiat
		state[coin].BaseValue = inbase
	}
	state[TOTAL].Balance = basetotal
	state[TOTAL].FiatValue = basetotal * FIATBTC

	// if first run and state not prev saved then mark our start position for statistical evaluation
	if _, ok := state[START]; !ok {
		state[START] = &coinstate{Coin: base, Balance: basebalance, Date: time.Now(), FiatValue: state[base].FiatValue}
	}

	///////////////////////////////////////////
	// print current balances

	Info.Print("BALANCES")
	Info.Printf("%v %v (%v %v) \n", base, fc(basebalance), fc(basebalance*FIATBTC), fiat)
	for _, coin = range targets {
		if state[coin].Balance > 0 {
			Info.Printf("%v %v (%v %v) \n", coin, fc(state[coin].Balance), fc(state[coin].FiatValue), fiat)
		}
	}
	Info.Printf("BALANCE Total %v %v over %v coins", fc(basetotal), base, fragmenttotal)

	///////////////////////////////////////////
	// start off by getting all our open orders (that have not been fullfilled for whatever reason) and cancel them
	Info.Println("CANCELLING ALL OPEN ORDERS")
	//
	if simulate == false {
		if ok := CancelAllOpenOrders(base, targets); !ok {
			Error.Println("Problem cancelling all open orders: bailing out.")
			return
		}
	}

	////////////////////////////////////////////
	// Analyse coins
	// for each coin get Analyse() to evaluate buy/sell and give a ranking of how strongly it is growing so we can prioritise
	if sss == false {
		Info.Println("ANALYSING DATA")

		for i, _ := range todo {
			action, ranking := Analyse(todo[i].coin)
			todo[i].action = action
			todo[i].ranking = ranking
		}
		// sort by ranking descending
		sort.Slice(todo, func(i, j int) bool { return todo[i].ranking > todo[j].ranking })
	}

	///////////////////////////////////////////
	// buying and selling for each coin
	// taking our analysis place our orders

	minbasetotrade := conf.GetFloat64("TradingRules.minbasetotrade")
	maxfragments := conf.GetFloat64("TradingRules.fragments")
	sales := 0
	if maxfragments == 0 {
		maxfragments = 1
	}

	for i, _ := range todo {
		coin = todo[i].coin
		coinbalance := state[coin].Balance
		action := todo[i].action
		basebalance := state[base].Balance

		if action == BUY && coinbalance == 0 {
			// check enough balance to make an order (minorder)
			// get current asking price
			if basebalance > minbasetotrade {
				throttle()
				Info.Println(coin + " Placing BUY  order")
				if fragmenttotal < maxfragments && basebalance > minbasetotrade*2 {
					fragmenttotal++
					basebalance = basebalance * (fragmenttotal / maxfragments)
				}
				Buy(base, coin, ticker[base+"_"+coin].Ask, basebalance)

			} else {
				Info.Printf("Balance of %v is lower (%v) than minbasetotrade rule (%v) Can't place buy order\n", base, fc(basebalance), fc(minbasetotrade))
			}
		}

		if action == SELL {
			// get current bidding price
			// get balance and sell all
			throttle()
			Info.Println(coin + " Placing SELL order")
			Sell(base, coin, ticker[base+"_"+coin].Bid, coinbalance)
			sales++
		}

		if action == NOACTION {

			Info.Print(coin + " Nothing to do")
		}
	}

	////////////////////////////////////
	//update state before saving
	Info.Print("UPDATING STATS")
	basetotal = state[base].Balance
	state[base].FiatValue = basetotal * FIATBTC
	state[base].BaseValue = basetotal
	s := fmt.Sprintf("coin|balance|BTC|%v\n", fiat)
	s += fmt.Sprintf("%v|%v|%v|%v\n", base, fc(basetotal), fc(basetotal), fn2(basetotal*FIATBTC))
	for _, coin = range targets {
		coinbalance = state[coin].Balance
		inbase := ticker[base+"_"+coin].Last * coinbalance
		basetotal += inbase
		state[coin].FiatValue = inbase * FIATBTC
		state[coin].BaseValue = inbase
		s += fmt.Sprintf("%v|%v|%v|%v\n", coin, fc(coinbalance), fc(inbase), fn2(inbase*FIATBTC))
	}
	state[TOTAL].Balance = basetotal
	state[TOTAL].FiatValue = basetotal * FIATBTC
	s += fmt.Sprintf("%v|%v|%v|%v\n", "TOTAL", fc(0), fc(basetotal), fn2(basetotal*FIATBTC))
	// what a hack!
	state[TOTAL].OrderNumber = s
	state[TOTAL].Misc = fmt.Sprintf("%v", time.Now().Sub(state[START].Date))
	////////////////////////////////////

	if sss {
		Info.Printf("SellSellSell run completed. %v sale(s) placed\n", sales)
	}
}

func getTargettedCoins() (targets []string) {

	targets = strings.Split(conf.GetString("Currency.targets"), ",")
	if len(targets) == 0 {
		targets = append(targets, conf.GetString("Currency.target"))
	}
	return
}
