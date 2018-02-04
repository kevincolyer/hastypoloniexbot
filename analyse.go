package main

import (
	"sort"
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
	Info.Printf("Analysis of %v using ema and sma for period of %v\n", pair, period)
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

	// extract and sort closes into a slice of vals
	sma := CalcSMA(c, smaperiods, 0)
	ema := CalcEMA(c, emaperiods)
	diff := ema - sma
	balance := state[coin].Balance
	last := state["LAST"].Coin
	advice = NOACTION
	anal := "ANAL "
	Info.Printf(anal+"Currently holding %v %v\n", fc(balance), coin)
	direction := coin
	if diff >= 0 {
		direction += " rising"
	} else {
		direction += " falling"
	}
	Info.Printf(anal+"%v: sma(%v): %v ema(%v): %v diff: %v\n", direction, smaperiods, fc(sma), emaperiods, fc(ema), fc(diff))

	if balance == 0 {
		// if last coin sold is this coin then do nothing (cooling off period)
		if coin == last {
			Info.Printf(anal+"Last coin sold %v is this coin - cooling off\n", last)
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
			Info.Printf(anal+"ema greater than sma but not by triggerbuy limit:%v %% (%v %%)\n", fp(ranking), fp(triggerbuy))
			return

		}
		advice = BUY
		Info.Printf(anal+"Recommend BUY %v ranking %v\n", coin, fp(ranking))
		return
	}

	purchaseprice := state[coin].PurchasePrice

	currentprice := c[0] // need a better indicator
	maxlosstorisk := conf.GetFloat64("TradingRules.maxlosstorisk")
	percentloss := (currentprice - purchaseprice) / currentprice
	if percentloss > 0 {
		percentloss = 0
	}
	// possible sell if trending down
	if balance > 0 && ema < sma {
		// curent price < purchase price-allowable loss the advice = sell
		if percentloss < 0 && -percentloss > maxlosstorisk {
			advice = SELL
			Info.Printf(anal+"Recommend SELL as loss %v %% is less than maxlosstorisk %v %%\n", fp(percentloss), fp(maxlosstorisk))
			return
		}
		if percentloss < 0 {
			Warning.Printf(anal+"Price is %v %% below purchase price but not at maxlosstorisk %v %%\n", fp(percentloss), fp(maxlosstorisk))
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
		Info.Printf(anal+"SELL: Coin is %v times greater than purchase price - triggered maxgrowth %v\n", fn(growth), fn(maxgrowth))
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
