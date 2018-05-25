package main

import (
	. "github.com/dex4er/go-tap"
	"testing"
	"time"
)

func TestAnalyse(t *testing.T) {
	b := NewBot()
	b.ConfInit("config")
	b.NewState()
	// store current time (used to fake time for training data)
	b.Now = time.Now()
	JustNow := b.Now.Add(-time.Minute * 5)
	Never := JustNow.Add(-time.Hour * 10000)

// 	d := AnalysisData{
// 		coin: "ETH",
// 
// 		emaperiods:    b.Conf.GetInt("Analysis.ema"),
// 		smaperiods:    b.Conf.GetInt("Analysis.sma"),
// 		triggerbuy:    b.Conf.GetFloat64("TradingRules.triggerbuy"),
// 		maxlosstorisk: b.Conf.GetFloat64("TradingRules.maxlosstorisk"),
// 		triggersell:   b.Conf.GetFloat64("TradingRules.triggersell"),
// 		maxgrowth:     b.Conf.GetFloat64("TradingRules.maxgrowth"),
// 
// 		ema:          96,
// 		sma:          90,
// 		currentprice: 98,
// 
// 		coinbalance:   0,
// 		lastcoin:      "STR",
// 		purchasedate:  Never,
// 		lastsold:      Never,
// 		lastema:       95,
// 		lastsma:       90,
// 		purchaseprice: 0,
// 	}
// 	d.cooloffduration, _ = time.ParseDuration(b.Conf.GetString("TradingRules.CoolOffDuration"))
// 
// 	CheckDuration(b.Now, b, &d)
        d:=newAnalysisData(b,JustNow,Never)
        
	Is(d.cooloffperiod, false, "Coin not held,Coin bought never, cool off false")
	Is(d.HeldForLongEnough, false, "Coin not held,HeldForLongEnough false")

	d.coinbalance = 1
	d.purchasedate = JustNow
	CheckDuration(b.Now, b, &d)
	Is(d.cooloffperiod, false, "Coin  held,Coin bought just now, cool off false")
	Is(d.HeldForLongEnough, false, "Coin held,Coin bought just now, HeldForLongEnough  false")

	d.purchasedate = Never
	CheckDuration(b.Now, b, &d)
	Is(d.cooloffperiod, false, "Coin  held,Coin bought ages ago, cool off false")
	Is(d.HeldForLongEnough, true, "Coin held,Coin bought ages ago, HeldForLongEnough  true")

	d.coinbalance = 0
	d.lastsold = JustNow
	CheckDuration(b.Now, b, &d)
	Is(d.cooloffperiod, true, "Coin  not held,Coin sold just now, cool off true")
	Is(d.HeldForLongEnough, false, "Coin not held,Coin sold just now, HeldForLongEnough  false")

	d.lastsold = Never
	CheckDuration(b.Now, b, &d)
	Is(d.cooloffperiod, false, "Coin  not held,Coin sold ages ago, cool off false")
	Is(d.HeldForLongEnough, false, "Coin not held,Coin sold ages ago, HeldForLongEnough  false")

	// reset
        d=newAnalysisData(b,JustNow,Never)
        

	// Test Analyse02
	d.analysisfunc = "02"
	b.LogInit("Testing Log")
	b.Logging = true

	// BUYING

	// test worth buying
	action, ranking := b.Analyse(d)
	Is(action, BUY, "Worth Buying: Coin is recommended buy(1)")

	// test ema and sma diffs correct
	Is(ranking, (d.ema-d.sma)/d.sma, "ema and sma diffs correct: Coin has correct percent diff ranking formula")

	//     test cant buy because held
	d.coinbalance = 1
	d.purchasedate = JustNow
	CheckDuration(b.Now, b, &d)
	action, ranking = b.Analyse(d)
	Is(action, NOACTION, "cant buy because held: Coin is recommended Noaction(0)")
	d.purchasedate = Never
	// 	CheckDuration(b.Now,b, &d)
	// 	action, ranking = b.Analyse(d)
	// 	Is(action, NOACTION, "cant buy because held 2: Coin is recommended Noaction(0)")
	d.coinbalance = 0
	//     test cant sell because not  (TODO can't test this!)

	//     test too soon to buy again
	d.lastsold = JustNow
	CheckDuration(b.Now, b, &d)
	action, ranking = b.Analyse(d)
	Is(action, NOACTION, "cant buy because just sold: Coin is recommended Noaction(0)")
	d.lastsold = Never
	CheckDuration(b.Now, b, &d)

	// NO ACTION
	// test not worth buying = no action
	d.ema = d.sma
	action, ranking = b.Analyse(d)
	Is(action, NOACTION, "Not worth buying: Coin is recommended NoAction(0)")

	//     test have bought but not ready to sell
	d.ema = d.sma - 5
	d.purchaseprice = 100
	d.purchasedate = JustNow
	CheckDuration(b.Now, b, &d)
	action, ranking = b.Analyse(d)
	Is(action, NOACTION, "Have bought but not ready to sell:Coin is recommended NoAction(0)")

        // RESET
        d=newAnalysisData(b,JustNow,Never)
        d.analysisfunc = "02"
        
	// SELL
	//     test should sell as have HeldForLongEnough
	d.coinbalance = 1
	d.purchaseprice = 80   // we bought low!
	d.purchasedate = Never // a long time ago
	CheckDuration(b.Now, b, &d)
	action, ranking = b.Analyse(d)
	Is(action, SELL, "should sell as have HeldForLongEnough:Coin is recommended SELL(2)")

	//      dont sell as we bought lower and are good time window
	d.ema = 96
	d.sma = 90
	d.purchasedate = JustNow // a long time ago
	CheckDuration(b.Now, b, &d)
	action, ranking = b.Analyse(d)
	Is(action, NOACTION, "dont sell as we bought lower and are good time window:Coin is recommended NOACTION")

        // RESET
        d=newAnalysisData(b,JustNow,Never)
        d.analysisfunc = "02"
        
	//     test should sell as making too much loss
	d.coinbalance = 1
	d.purchaseprice = 98
	d.currentprice = 95
	d.sma = 98
	d.ema = 96
	action, ranking = b.Analyse(d)
	Is(action, SELL, "should sell as making too much loss(maxlosstorisk and EMA/SMA trending down):Coin is recommended SELL")
	d.currentprice = 97
	action, ranking = b.Analyse(d)
	Is(action, SELL, "should sell as making too much loss(less than price but not yet maxlosstorisk but ema/sma trending down):Coin is recommended SELL")
	d.currentprice = 98
	action, ranking = b.Analyse(d)
	Is(action, NOACTION, "should NOT sell as coin steady but ema/sma trending down:Coin is recommended NOACTION")

	// test ranking is accurate

	DoneTesting()
}


func newAnalysisData(b *Bot, JustNow, Never time.Time) (d AnalysisData) {
    
        d = AnalysisData{
		coin: "ETH",

		emaperiods:    b.Conf.GetInt("Analysis.ema"),
		smaperiods:    b.Conf.GetInt("Analysis.sma"),
		triggerbuy:    b.Conf.GetFloat64("TradingRules.triggerbuy"),
		maxlosstorisk: b.Conf.GetFloat64("TradingRules.maxlosstorisk"),
		triggersell:   b.Conf.GetFloat64("TradingRules.triggersell"),
		maxgrowth:     b.Conf.GetFloat64("TradingRules.maxgrowth"),

		ema:          96,
		sma:          90,
		currentprice: 98,

		coinbalance:   0,
		lastcoin:      "STR",
		purchasedate:  Never,
		lastsold:      Never,
		lastema:       95,
		lastsma:       90,
		purchaseprice: 0,
	}
	d.cooloffduration, _ = time.ParseDuration(b.Conf.GetString("TradingRules.CoolOffDuration"))

	CheckDuration(b.Now, b, &d)
        return
}
