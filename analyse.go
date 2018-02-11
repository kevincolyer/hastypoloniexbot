package main

import (
	"sort"
	"time"
	//         "fmt"

	"gitlab.com/wmlph/poloniex-api"
)

func CalcEMA(closes []float64, periods int) (v float64) {
	//     Initial SMA: 10-period sum / 10
	//     Multiplier: (2 / (Time periods + 1) ) = (2 / (10 + 1) ) = 0.1818 (18.18%)
	//     EMA: {Close - EMA(previous day)} x multiplier + EMA(previous day).
	v = CalcSMA(closes, periods, periods+1)
	mult := 2.0 / (float64(periods + 1)) // the traditional form
	// mult:= 1/float64(periods) // wilder form
	//     fmt.Printf("mult=%v, initsma=%v\n",mult,v)
	for i := periods; i > 0; i-- {
		v = (closes[i]-v)*mult + v
	}
	return
}

// optional offset is to provide an sma to start ema func.
func CalcSMA(closes []float64, periods, offset int) (v float64) {
	v = 0
	j := 0.0
	for i := offset; i <= offset+periods; i++ {
		v += closes[i]
		j++
	}
	v = v / j
	return
}

func Analyse(coin string) (advice int, ranking float64) {
	advice = NOACTION
	pair := conf.GetString("Currency.Base") + "_" + coin
	period := conf.GetInt("Analysis.period")
	Info.Printf(coin+" Analysis using ema and sma for period of %v\n", period)
	// get chartdata from polo for coin
	data, err := exchange.ChartDataPeriod(pair, period)
	if err != nil {
		Warning.Printf("Could not retrieve data for pair %s. Error %v\n", pair, err)
		return
	}
	closings := mungeCoinChartData(data)
	return analyseChartData(closings, coin)
}

func analyseChartData(c []float64, coin string) (advice int, ranking float64) {

	ranking = 0
	emaperiods := conf.GetInt("Analysis.ema")
	smaperiods := conf.GetInt("Analysis.sma")
	advice = NOACTION
	anal := coin + " "
	sma := CalcSMA(c, smaperiods, 0)
	ema := CalcEMA(c, emaperiods)
	diff := ema - sma
	balance := state[coin].Balance
	last := state[LAST].Coin
	cooloffperiod := false
	lastsold := state[coin].SaleDate

	dur, err := time.ParseDuration(conf.GetString("TradingRules.CoolOffDuration"))
	if err != nil {
		dur, _ = time.ParseDuration("2h")
		Warning.Printf("Couldn't parse CoolOffDuration (%v). Setting default to %v\n", conf.GetString("TradingRules.CoolOffDuration"), dur)
	}
	if balance == 0 && lastsold.After(time.Now().Add(-dur)) {
		cooloffperiod = true
		//		Info.Print(anal + "is in cooling off period")

	} /*else {
		Info.Print(anal + "is NOT in cooling off period")
	}*/

	if balance > 0 {
		Info.Printf(anal+"Currently holding %v\n", fc(balance))
	} else {
		Info.Printf(anal + "Currently not holding coin\n")
	}

	direction := coin
	if diff >= 0 {
		direction += " ema/sma +ve"
	} else {
		direction += " ema/sma -ve"
	}
	Info.Printf(anal+"%v: sma(%v): %v ema(%v): %v diff: %v\n", direction, smaperiods, fc(sma), emaperiods, fc(ema), fc(diff))

	if balance == 0 {
		// if last coin sold is this coin then do nothing (cooling off period)
		if cooloffperiod {
			Info.Printf(anal+" in cooling off period. Not Buying.\n", last)
			return
		}
		// if ema<sma advice nothing return
		if ema < sma {
			Info.Printf(anal + "ema is less than sma - coin trending down not a good buy\n")
			return
		}
		triggerbuy := conf.GetFloat64("TradingRules.triggerbuy")
		ranking = diff / sma
		if ema > sma && ranking < triggerbuy {
			Info.Printf(anal+"ema greater than sma but not by triggerbuy limit:%v %% (%v %%)\n", fp2(ranking), fp2(triggerbuy))
			return

		}
		Info.Printf(anal+"Recommend BUY ranking %v above triggerbuy %v\n", fp2(ranking), fp2(triggerbuy))
		advice = BUY // only recommended as  balance ==0
		return
	}

	purchaseprice := state[coin].PurchasePrice

	currentprice := c[0] // TODO need a better indicator
	maxlosstorisk := conf.GetFloat64("TradingRules.maxlosstorisk")
	triggersell := conf.GetFloat64("TradingRules.triggersell")
	percentloss := (currentprice - purchaseprice) / purchaseprice
	if percentloss > 0 {
		percentloss = 0
	}
	// sell if trending down (buy back should be delayed a few hours)
	Info.Printf(anal+"PurchasePrice %v currentprice %v percentloss %v purchasedate %v lastsale %v \n", fc(purchaseprice), fc(currentprice), fp2(percentloss), state[coin].Date, state[coin].SaleDate)

	if balance > 0 && percentloss < 0 {

		if -percentloss < maxlosstorisk {
			Warning.Printf(anal+"Price is %v %% below purchase price but not at maxlosstorisk %v %%\n", fp2(percentloss), fp2(maxlosstorisk))
			return
		}
		advice = SELL
		Info.Printf(anal+"Price is %v %% below purchase price and greater than maxlosstorisk %v %%. Advice SELL\n", fp2(percentloss), fp2(maxlosstorisk))
		return
	}
	// 		if balance > 0 &&  currentprice < purchaseprice {
	// 			advice = SELL
	// 			Info.Printf(anal+"Recommend SELL as currentprice %v is less than purchased price %v\n", fc(currentprice), fc(purchaseprice))
	// 			return
	// 		}
	// ma diff is lower than triggersell
	if balance > 0 && diff/sma < triggersell {
		advice = SELL
		Info.Printf(anal+"Recommend SELL as ema-sma/sma %v is less than triggersell %v\n", fp(diff/sma), fp(triggersell))
		return
	}
	if balance > 0 && ema < sma {
		// coin is trending down in value
		// TODO CARE NEEDED HERE!
		// curent price < purchase price-allowable loss the advice = sell
		if percentloss < 0 && -percentloss > maxlosstorisk {
			advice = SELL
			Info.Printf(anal+"Recommend SELL as loss %v %% is less than maxlosstorisk %v %%\n", fp2(percentloss), fp2(maxlosstorisk))
			return
		}
		// current price > purchase price info - keep - coin is growing in value
		if percentloss == 0 {
			Warning.Printf(anal + "Coin is in profit and growing in value but trending down")
			return
		}
		// current price > purchase price + max allowed growth - sell (get out on top)

	}
	maxgrowth := conf.GetFloat64("TradingRules.maxgrowth")
	growth := currentprice - purchaseprice/purchaseprice
	if balance > 0 && growth > maxgrowth {
		Info.Printf(anal+"SELL:  %v times greater than purchase price - triggered maxgrowth %v\n", fn(growth), fn(maxgrowth))
		advice = SELL
		return
	}
	if balance > 0 && state[coin].Date.Before(time.Now().Add(-time.Hour*22)) {
		Info.Printf(anal+"SELL: Purchased more than 22 hours ago %v\n", state[coin].Date)
		advice = SELL
		return
	}
	Info.Print(anal + "Nothing to do. No concerns")
	return
}

func mungeCoinChartData(data poloniex.ChartData) (closings []float64) {
	sort.Slice(data, func(i, j int) bool { return data[i].Date > data[j].Date }) // descending sort ie. now back into past
	for i, _ := range data {
		closings = append(closings, data[i].Close)
	}
	return
}

// func getJsonCoinChart(coin string) (json string, err error) {
//     return
// }
