package main

import (
	"sort"
	"time"
	//         "fmt"

	"gitlab.com/wmlph/poloniex-api"
)

func CalcEMA(closes []float64, periods int) (ema float64) {
	// note closes is sorted in reverse order, with current at 0 and 50th prev data point at 49

	//     Initial SMA: 10-period sum / 10
	//ema = CalcSMA(closes, periods, periods+1)
	ema = CalcSMA(closes, 10, periods+10+1)

	//     Multiplier: (2 / (Time periods + 1) ) = (2 / (10 + 1) ) = 0.1818 (18.18%)
	mult := 2.0 / (float64(periods + 1)) // the traditional form // mult:= 1/float64(periods) // wilder form

	//     EMA: {Close - EMA(previous day)} x multiplier + EMA(previous day).
	for i := periods; i > 0; i-- {
		ema = (closes[i]-ema)*mult + ema
	}
	return
}

// optional offset is to provide an sma to start ema func.
func CalcSMA(closes []float64, periods, offset int) (sma float64) {
	j := 0.0
	// closes is sorted in reverse order but that is not needed for sma function
	for i := offset; i <= offset+periods; i++ {
		sma += closes[i]
		j++
	}
	sma = sma / j
	return
}

func Analyse(coin string) (advice int, ranking float64) {
	advice = NOACTION
	pair := conf.GetString("Currency.Base") + "_" + coin
	period := conf.GetInt("Analysis.period")
	if Logging {
		Info.Printf(coin+" Analysis using ema and sma for period of %v\n", period)
	}
	// get chartdata from polo for coin
	data, err := exchange.ChartDataPeriod(pair, period)
	if err != nil {
		if Logging {
			Warning.Printf("Could not retrieve data for pair %s. Error %v\n", pair, err)
		}
		return
	}
	closings := mungeCoinChartData(data)

	d := analysisdata{coin: coin}

	d.emaperiods = conf.GetInt("Analysis.ema")
	d.smaperiods = conf.GetInt("Analysis.sma")
	d.sma = CalcSMA(closings, d.smaperiods, 0)
	d.ema = CalcEMA(closings, d.emaperiods)

	d.coinbalance = state[coin].Balance
	d.lastcoin = state[LAST].Coin
	d.purchasedate = state[coin].Date
	d.lastsold = state[coin].SaleDate
	d.lastema = state[coin].LastEma
	d.lastsma = state[coin].LastSma
	state[coin].LastEma = d.ema
	state[coin].LastSma = d.sma
	d.cooloffduration, _ = time.ParseDuration(conf.GetString("TradingRules.CoolOffDuration"))
	// 	if err != nil {
	// 		dur, _ = time.ParseDuration("2h")
	// 		if Logging {
	// 			Warning.Printf("Couldn't parse CoolOffDuration (%v). Setting default to %v\n", conf.GetString("TradingRules.CoolOffDuration"), dur)
	// 		}
	// 	}
	if d.coinbalance == 0 && d.lastsold.After(time.Now().Add(-d.cooloffduration)) {
		d.cooloffperiod = true
	}

	d.triggerbuy = conf.GetFloat64("TradingRules.triggerbuy")
	d.purchaseprice = state[coin].PurchasePrice
	d.currentprice = closings[0] // TODO need a better indicator
	d.maxlosstorisk = conf.GetFloat64("TradingRules.maxlosstorisk")
	d.triggersell = conf.GetFloat64("TradingRules.triggersell")
	d.maxgrowth = conf.GetFloat64("TradingRules.maxgrowth")
	d.HeldForLongEnough = d.purchasedate.After(time.Now().Add(-time.Hour * 22)) // yuk

	return analyseChartData(d)
}

type analysisdata struct {
	//	advice          int
	//	ranking         float64
	ema               float64
	sma               float64
	emaperiods        int
	smaperiods        int
	coin              string
	coinbalance       float64
	purchaseprice     float64
	purchasedate      time.Time
	lastsold          time.Time
	lastcoin          string
	cooloffperiod     bool
	HeldForLongEnough bool
	cooloffduration   time.Duration
	currentprice      float64
	maxholdduration   bool
	lastema           float64
	lastsma           float64
	triggerbuy        float64
	triggersell       float64
	maxlosstorisk     float64
	percentloss       float64
	maxgrowth         float64
}

func analyseChartData(d analysisdata) (advice int, ranking float64) {
	ranking = 0
	diff := d.ema - d.sma
	advice = NOACTION
	anal := d.coin + " "
	trendingdown := pdiff(d.ema, d.sma) < pdiff(d.lastema, d.lastsma)

	if Logging {
		if trendingdown {
			Info.Printf(anal+"ema diff %v is trending down from last diff %v\n", fc(pdiff(d.ema, d.sma)), fc(pdiff(d.lastema, d.lastsma)))
		} else {
			Info.Printf(anal+"ema diff %v is trending up from last diff %v\n", fc(pdiff(d.ema, d.sma)), fc(pdiff(d.lastema, d.lastsma)))
		}
	}

	if Logging {
		if d.coinbalance > 0 {
			Info.Printf(anal+"Currently holding %v\n", fc(d.coinbalance))
		} else {
			Info.Printf(anal + "Currently not holding coin\n")
		}
		direction := d.coin
		if diff >= 0 {
			direction += " ema/sma +ve"
		} else {
			direction += " ema/sma -ve"
		}
		Info.Printf(anal+"%v: sma(%v): %v ema(%v): %v diff: %v\n", direction, d.smaperiods, fc(d.sma), d.emaperiods, fc(d.ema), fc(diff))
	}

	if d.coinbalance == 0 {
		// if last coin sold is this coin then do nothing (cooling off period)
		if d.cooloffperiod {
			if Logging {
				Info.Printf(anal+" in cooling off period. Not Buying.\n", d.lastcoin)
			}
			return
		}
		// if ema<sma advice nothing return
		if d.ema < d.sma {
			if Logging {
				Info.Printf(anal + "ema is less than sma - coin trending down not a good buy\n")
			}
			return
		}
		ranking = diff / d.sma
		if d.ema > d.sma && ranking < d.triggerbuy {
			if Logging {
				Info.Printf(anal+"ema greater than sma but not by triggerbuy limit:%v %% (%v %%)\n", fp2(ranking), fp2(d.triggerbuy))
			}
			return

		}
		// ema>sma by triggerbuy...
		if Logging {
			Info.Printf(anal+"Recommend BUY ranking %v above triggerbuy %v\n", fp2(ranking), fp2(d.triggerbuy))
		}
		advice = BUY // only recommended as  coinbalance ==0
		return
	}

	percentloss := (d.currentprice - d.purchaseprice) / d.purchaseprice
	pl := percentloss
	if percentloss > 0 {
		percentloss = 0
	}
	// sell if trending down (buy back should be delayed a few hours)
	if Logging {
		Info.Printf(anal+"PurchasePrice %v currentprice %v percentloss %v %v purchasedate %v lastsale %v \n", fc(d.purchaseprice), fc(d.currentprice), fn(percentloss), fn(pl), d.purchasedate, d.lastsold)
	}

	// 	if coinbalance > 0 && percentloss < 0 {
	//
	// 		if -percentloss < maxlosstorisk {
	// 			if Logging { Warning.Printf(anal+"Price is %v %% below purchase price but not at maxlosstorisk %v %%\n", fp2(percentloss), fp2(maxlosstorisk)) }
	// 			return
	// 		}
	// 		advice = SELL
	// 		if Logging { Info.Printf(anal+"Price is %v %% below purchase price and greater than maxlosstorisk %v %%. Advice SELL\n", fp2(percentloss), fp2(maxlosstorisk)) }
	// 		return
	// 	}
	// 		if coinbalance > 0 &&  currentprice < purchaseprice {
	// 			advice = SELL
	// 			if Logging { Info.Printf(anal+"Recommend SELL as currentprice %v is less than purchased price %v\n", fc(currentprice), fc(purchaseprice)) }
	// 			return
	// 		}
	// ma diff is lower than triggersell
	if d.coinbalance > 0 && diff/d.sma < d.triggersell {
		advice = SELL
		if Logging {
			Info.Printf(anal+"Recommend SELL as ema-sma/sma %v is less than triggersell %v\n", fp(diff/d.sma), fp(d.triggersell))
		}
		return
	}
	if d.coinbalance > 0 && d.ema < d.sma {
		// coin is trending down in value
		// TODO CARE NEEDED HERE!
		// curent price < purchase price-allowable loss the advice = sell
		if percentloss < 0 && -percentloss > d.maxlosstorisk {
			advice = SELL
			if Logging {
				Info.Printf(anal+"Recommend SELL as loss %v %% is less than maxlosstorisk %v %%\n", fp2(percentloss), fp2(d.maxlosstorisk))
			}
			return
		}
		// current price > purchase price info - keep - coin is growing in value
		if percentloss == 0 {
			if Logging {
				Warning.Printf(anal + "Coin is in profit and growing in value but trending down")
			}
			return
		}
		// current price > purchase price + max allowed growth - sell (get out on top)

	}
	growth := (d.currentprice - d.purchaseprice) / d.purchaseprice
	if d.coinbalance > 0 && growth > d.maxgrowth {
		if Logging {
			Info.Printf(anal+"SELL:  %v times greater than purchase price - triggered maxgrowth %v\n", fn(growth), fn(d.maxgrowth))
		}
		advice = SELL
		return
	}
	if d.coinbalance > 0 && d.HeldForLongEnough {
		if Logging {
			Info.Printf(anal + "SELL: Held for long enough threshold exceeded.\n")
		}
		advice = SELL
		return
	}
	if Logging {
		Info.Print(anal + "Nothing to do. No concerns")
	}
	return
}

func mungeCoinChartData(data poloniex.ChartData) (closings []float64) {
	sort.Slice(data, func(i, j int) bool { return data[i].Date > data[j].Date }) // descending sort ie. now back into past
	for i, _ := range data {
		closings = append(closings, data[i].Close)
	}
	return
}

func pdiff(e, s float64) float64 {
	if s == 0 {
		return 0
	}
	return (e - s) / s
}
