package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gitlab.com/wmlph/poloniex-api"
)

func (b *Bot) Trade() {
	/////////////////////////////////////
	// this can be used in training or trading mode. Different set up for the analysers are used
	// in that case and some reporting is not required

	// note in Training mode b.Ticker will be initialised by b.Train() before calling to fill it with
	// the precached training data. At the moment Bid, Ask and Last are required.

	/////////////////////////////////////
	// Get poloniex data and set up variables from config file
	if b.Training == false {
		b.LogInfo("GETTING POLONIEX DATA")

		b.Exchange = poloniex.NewKS(
			b.Conf.GetString("Credentials.apikey"),
			b.Conf.GetString("Credentials.secret")) // check for failure needed here?
	}

	var coinbalance float64
	var fragmenttotal float64 // how many coins we will split our base coin into
	var coin string
	var todo []coinaction
	var balances poloniex.AvailableAccountBalances
	var err error

	// Set up some variables from config. Just for convenience and readability
	fiat := b.Conf.GetString("Currency.Fiat")
	base := b.Conf.GetString("Currency.Base")
	sss := b.Conf.GetBool("BotControl.SellSellSell")
	simulate := b.Conf.GetBool("BotControl.Simulate")

	// just in case there are some confusing flags set in config file.
	if b.Training == true {
		sss = false
		simulate = true
	} else {
		// Get Ticker {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
		b.Ticker, err = b.Exchange.Ticker()
		if err != nil {
			b.LogErrorf("Fatal error getting ticker data from poloniex: %v", err)
			return
		}
	}
	// must come after filling ticker with data.
	FIATBTC := b.Ticker[fiat+"_BTC"].Last // TODO can be other curency than usdt

	//////////////////////////////////////////////////
	// Get list of coins we are targetting
	// b.Conf.GetString("Currency.targets")
	targets := b.GetTargettedCoins()
	sort.Strings(targets)

	///////////////////////////////////////////
	// start off by getting all our open orders (that have not been fullfilled for whatever reason) and cancel them
	//
	if simulate != true {
		b.LogInfo("CANCELLING ALL OPEN ORDERS")
		if ok := b.CancelAllOpenOrders(base, targets); !ok {
			b.LogError("Problem cancelling all open orders: bailing out.")
			return
		}
		// TODO there may be coins cancelling or not yet canceled that will mess up the balance calcs!
		// if not simulating! get balance of base currency
		balances, err = b.Exchange.AvailableAccountBalances() // kpc added this function
		if err != nil {
			b.LogErrorf("Failed to get coin AccountBalances from poloniex: %v", err)
		}
		b.State[base].Balance = balances.Exchange[base]
		// + balances[base].OnOrders // include open buy orders - they will get cancelled below. Posssible race condition here!
	}

	// more variables now we know them
	basebalance := b.State[base].Balance // SIMULATION
	basetotal := basebalance
	b.State[base].FiatValue = basebalance * FIATBTC
	b.State[base].BaseValue = basebalance // for completeness

	// populate b.state with data
	for _, coin = range targets {
		// if we have not loaded this coin from json, we need to add it to the map.
		if _, ok := b.State[coin]; !ok {
			b.State[coin] = &coinstate{Coin: coin, Balance: 0.0}
		}

		if simulate {
			coinbalance = b.State[coin].Balance // SIMULATION!
		} else {
			coinbalance = balances.Exchange[coin] // REAL THING!
		}
		infiat := b.Ticker[base+"_"+coin].Last * coinbalance * FIATBTC
		inbase := b.Ticker[base+"_"+coin].Last * coinbalance

		action := NOACTION
		if coinbalance > 0 {
			fragmenttotal++
			basetotal += inbase
			if sss == true {
				action = SELL
			}
		}
		todo = append(todo, coinaction{Coin: coin, Action: action}) // used to prepare todo slice for use in sellsellsell
		b.State[coin].Balance = coinbalance
		b.State[coin].FiatValue = infiat
		b.State[coin].BaseValue = inbase
	}
	b.State[TOTAL].Balance = basetotal
	b.State[TOTAL].FiatValue = basetotal * FIATBTC

	// if first run and state not prev saved then mark our start position for statistical evaluation
	if _, ok := b.State[START]; !ok {
		b.State[START] = &coinstate{Coin: base, Balance: basebalance, Date: getTimeNow(), FiatValue: b.State[base].FiatValue}
	}

	////////////////////////////////////////////
	// Print current balances to log
	if b.Training == false {
		b.LogInfo("BALANCES (PROVISIONAL - ORDERS PENDING/CANCELLING)")
		b.LogInfof("%v %v (%v %v) ", base, fc(basebalance), fc(basebalance*FIATBTC), fiat)
		for _, coin = range targets {
			if b.State[coin].Balance > 0 {
				b.LogInfof("%v %v (%v %v) ", coin, fc(b.State[coin].Balance), fc(b.State[coin].FiatValue), fiat)
			}
		}
		b.LogInfof("BALANCE Total %v %v over %v coins", fc(basetotal), base, fragmenttotal)
	}

	////////////////////////////////////////////
	// Analyse coins
	// for each coin get Analyse() to evaluate buy/sell and give a ranking of how strongly it is growing so we can prioritise
	if sss == false && b.Training == false {
		b.LogInfo("ANALYSING DATA")

		for i, _ := range todo {
			action, ranking := b.Analyse(todo[i].Coin)
			todo[i].Action = action
			todo[i].Ranking = ranking
		}
		// sort by ranking descending (largest first)
		sort.Slice(todo, func(i, j int) bool { return todo[i].Ranking > todo[j].Ranking })
	}

	if b.Training == true {
		// TODO analysis setup for training...
	}

	////////////////////////////////////
	// Place orders
	b.PlaceBuyAndSellOrders(base, fragmenttotal, todo)

	basetotal = b.State[base].Balance
	b.State[base].FiatValue = basetotal * FIATBTC
	b.State[base].BaseValue = basetotal

	////////////////////////////////////
	// Update state before saving
	// TODO Pause here and perhaps await an update?
	// this is just for reporting purposes, to provide data for a web page updated by perl script report.pl

	if b.Training == false {
		b.LogInfo("UPDATING STATS")
		s := fmt.Sprintf("coin|balance|BTC|%v|held\n", fiat)
		s += fmt.Sprintf("%v|%v|%v|%v|-\n", base, fc(basetotal), fc(basetotal), fn2(basetotal*FIATBTC))

		for _, coin = range targets {
			coinbalance = b.State[coin].Balance
			inbase := b.Ticker[base+"_"+coin].Last * coinbalance
			basetotal += inbase
			b.State[coin].FiatValue = inbase * FIATBTC
			b.State[coin].BaseValue = inbase
			dur := "-"
			if !b.State[coin].Date.IsZero() && coinbalance > 0 {
				dur = getTimeNow().Sub(b.State[coin].Date).String()
			}
			s += fmt.Sprintf("%v|%v|%v|%v|%v\n", coin, fc(coinbalance), fc(inbase), fn2(inbase*FIATBTC), dur)
		}

		b.State[TOTAL].Balance = basetotal
		b.State[TOTAL].FiatValue = basetotal * FIATBTC
		s += fmt.Sprintf("%v|%v|%v|%v|-\n", "TOTAL", fc(0), fc(basetotal), fn2(basetotal*FIATBTC))
		// what a hack!
		b.State[TOTAL].OrderNumber = s
		b.State[TOTAL].Misc = fmt.Sprintf("%v", getTimeNow().Sub(b.State[START].Date))
	}
}

func (b *Bot) GetTargettedCoins() (targets []string) {
	targets = strings.Split(b.Conf.GetString("Currency.targets"), ",")
	if len(targets) == 0 {
		targets = append(targets, b.Conf.GetString("Currency.target"))
	}
	return
}

// buying and selling for each coin
// using analysis to place our orders
func (b *Bot) PlaceBuyAndSellOrders(base string, fragmenttotal float64, todo []coinaction) {
	///////////////////////////////////////////
	b.LogInfo("PLACING ORDERS")
	minbasetotrade := b.Conf.GetFloat64("TradingRules.minbasetotrade")
	maxfragments := b.Conf.GetFloat64("TradingRules.fragments")
	sales := 0

	for i, _ := range todo {
		coin := todo[i].Coin

		coinbalance := b.State[coin].Balance
		action := todo[i].Action
		basebalance := b.State[base].Balance

		if action == BUY && coinbalance == 0 {
			// check enough balance to make an order (minorder)
			// get current asking price
			if basebalance > minbasetotrade {
				Throttle()
				b.LogInfo(coin + " Placing BUY  order")
				// TODO need to figure out fragments better - especially if an order does not sell or buy!!!
				if fragmenttotal < maxfragments && basebalance > minbasetotrade*2 {
					fragmenttotal++
					basebalance = basebalance * (fragmenttotal / maxfragments)
				}
				b.Buy(base, coin, b.Ticker[base+"_"+coin].Ask, basebalance)

			} else {
				b.LogInfof(coin+" buy: Can't place buy order - balance of %v is lower (%v) than minbasetotrade rule (%v)", base, fc(basebalance), fc(minbasetotrade))
			}
		}

		if action == SELL {
			// get current bidding price
			// get balance and sell all
			Throttle()
			b.LogInfo(coin + " Placing SELL order")
			b.Sell(base, coin, b.Ticker[base+"_"+coin].Bid, coinbalance)
			sales++
		}

		if action == NOACTION {
			b.LogInfo(coin + " Nothing to do")
		}
	}
}

func getTimeNow() time.Time {
	return time.Now()
}
func getTimeNowString() (now string) {
	return time.Now().Format("2006/01/02 15:04:05")
}
