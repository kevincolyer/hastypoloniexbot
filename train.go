package main

import (
	"encoding/csv"
	// 	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	// 	"sort"
	"gitlab.com/wmlph/poloniex-api"
	"strconv"
	"strings"
	"time"
)

func (b *Bot) Train(traincoins string) {

	b.LogInfo("Loading training data")
	b.MyTrainingData = b.loadPreparedData()

	// Setup -----------------------------------------------------

	// open a csv file for dumping data
	// open data directory
	_, err := ioutil.ReadDir(b.TrainingDataDir)
	if err != nil {
		b.LogErrorf("Fatal error reading data directory: %v (is it created?)", err)
		return
	}

	b.TrainingParams = strings.ToLower(b.TrainingParams)

	file := b.TrainingOutputDir + "/" + fmt.Sprintf("%d", time.Now().Unix()) + "-" + strings.Replace(b.TrainingParams, "=", "-", -1) + ".csv" // with git commit ? with latest time ?//TODO Bug here?

	fmt.Println(file)
	f, err := os.Create(file)
	if err != nil {
		panic(fmt.Errorf("Fatal error opening file for writing: %s \n", err))
	}
	defer f.Close()

	w := csv.NewWriter(f)
	records := [][]string{
		{"triggerbuy", "triggersell", "buys", "sells", "profit"},
	}

	//targets=config or comma sep list config. default All
	if traincoins == "" {
		traincoins = "all"
	}
	switch traincoins {
	case "all":
		s := ""
		for pairs, _ := range b.MyTrainingData {
			if !strings.HasPrefix(string(pairs), "BTC") {
				continue
			}
			s += strings.TrimPrefix(string(pairs), "BTC_") + ","
		}
		b.Conf.Set("Currency.targets", strings.TrimSuffix(s, ","))
	case "config":
		break
	default:
		b.Conf.Set("Currency.targets", traincoins)
	}
	fmt.Println("Using these currencies:", b.Conf.GetString("Currency.targets"))
	targets := strings.Split(b.Conf.GetString("Currency.targets"), ",")
	p := make(map[string]string)
	for _, value := range strings.Split(b.TrainingParams, ",") {
		kv := strings.Split(value, "=")
		p[kv[0]] = kv[1]
	}
	fmt.Println("Using these params: ", p)
	// ------------------------------------------------------------
	// prep for MAIN LOOP
	// TODO use go routines to speed things on? Depends how slow!
	var tickerpairs []Pair
	for keys, _ := range b.MyTrainingData {
		tickerpairs = append(tickerpairs, keys) // b.Ticker has strings as keys
	}

	b.Logging = false
	b.LogInfo("You wont see this!")

	lengthTrainingData := len(b.MyTrainingData["USDT_BTC"])
	// defaults
	tbLo := 0.005
	tbHi := 0.095
	steps := 10
	lim := 20 * 24 // one day's worth of 5 min intervals
	//lim*=7

	// Overrides from TrainingParams
	for key, value := range p {
		switch key {
		case "tblo":
			tbLo, err = strconv.ParseFloat(value, 64)
		case "tbhi":
			tbHi, err = strconv.ParseFloat(value, 64)
		case "steps":
			steps, err = strconv.Atoi(value)
		case "lim":
			lim, err = strconv.Atoi(value)
                case "af":
                        cacheAnalysisFunc=value
		}
	}
	//cache some values for now
	cacheCooloffduration, _ = time.ParseDuration(b.Conf.GetString("TradingRules.CoolOffDuration"))
        minbasetotrade := b.Conf.GetFloat64("TradingRules.minbasetotrade")
        
	// setup variables for loop
	tbSteps := (tbHi - tbLo) / float64(steps)
	if lim > 0 {
		lengthTrainingData = lim
	} // truncate it for speed in debugging
	permutations := lengthTrainingData * steps * steps * (steps + 1) / 2 // 1/2n(n+1)
	b.Ticker = make(poloniex.Ticker)
	maxprofit := -1.0
	counter := 1
	// MAIN LOOP
	for tb := tbLo; tb < tbHi; tb += tbSteps { //38
		b.Conf.Set("TradingRules.triggerbuy", tb)
		for ts := 0.0; ts < tb; ts += tbSteps {
			b.Conf.Set("TradingRules.triggersell", ts)
			//-----------------------------------------------------------
			b.NewState()
			rand.Seed(1970) // a good year

			buys := 0
			sells := 0
			initialBalance := b.State["BTC"].Balance // profit or loss

			// loop training data
			for tick := 0; tick < lengthTrainingData; tick++ {
				counter++
				fmt.Printf("\rCalculating %v of %v ", CommaInt(counter), CommaInt(permutations))
				b.TrainingDataTick = tick
				// Load Ticker
				//      prepare ticker - include USDT_BTC as well as coins
				for _, value := range tickerpairs {
					b.Ticker[string(value)] = poloniex.TickerEntry{
						Ask:  b.MyTrainingData[value][tick].Ask,
						Last: b.MyTrainingData[value][tick].Last,
						Bid:  b.MyTrainingData[value][tick].Bid,
					}
				}
				b.Ticker["USDT_BTC"] = poloniex.TickerEntry{
					Ask:  b.MyTrainingData["USDT_BTC"][tick].Ask,
					Last: b.MyTrainingData["USDT_BTC"][tick].Last,
					Bid:  b.MyTrainingData["USDT_BTC"][tick].Bid,
				}

				// Run trade for this moment...
				bb, ss := b.Trade()
				buys += bb
				sells += ss
				// shortcut end if run out of currency to trade with
				if buys==sells && b.State["BTC"].Balance<=minbasetotrade {
                                    counter+=lengthTrainingData-tick
                                    break
                                }
			} // end tick loop

			// Get bitcoin value of all trades
			baseTotal := b.State["BTC"].Balance
			for _, coin := range targets {
				coinBalance := b.State[coin].Balance
				baseTotal += b.Ticker["BTC_"+coin].Last * coinBalance
			}
			profit := baseTotal - initialBalance
			// calc profit or loss
			if profit > maxprofit {
				fmt.Printf("\ntb:%v, ts:%v, buys:%v, sells:%v, profit:%v\n", tb, ts, buys, sells, fc(baseTotal-initialBalance))
				maxprofit = profit
			}
			r := strings.Split(fmt.Sprintf("%v %v %v %v %v", tb, ts, buys, sells, baseTotal-initialBalance), " ")

			//  stuff into CSV the profit, buys and sells for parameters varied and analysis chosen
			records = append(records, r)
		} // end ts
	} //end tb

	//----------------------------------------------------------
	// complete CSV
	fmt.Println()
	b.Logging = true
	b.LogInfo("Writing results file: " + file)
	w.WriteAll(records) // calls Flush internally
	if err := w.Error(); err != nil {
		panic(fmt.Errorf("error writing csv:", err))
	}
}

func (b *Bot) TrainPrepAnalysisData(coin string) AnalysisData {
	pair := Pair(b.Conf.GetString("Currency.Base") + "_" + coin)
	tick := b.TrainingDataTick

	d := AnalysisData{
		coin:          coin,
		emaperiods:    b.Conf.GetInt("Analysis.ema"),
		smaperiods:    b.Conf.GetInt("Analysis.sma"),
		triggerbuy:    b.Conf.GetFloat64("TradingRules.triggerbuy"),
		maxlosstorisk: b.Conf.GetFloat64("TradingRules.maxlosstorisk"),
		triggersell:   b.Conf.GetFloat64("TradingRules.triggersell"),
		maxgrowth:     b.Conf.GetFloat64("TradingRules.maxgrowth"),
		sma:           b.MyTrainingData[pair][tick].Sma50,
		ema:           b.MyTrainingData[pair][tick].Ema30,
		currentprice:  b.MyTrainingData[pair][tick].Last,
		coinbalance:   b.State[coin].Balance,
		lastcoin:      b.State[LAST].Coin,
		purchasedate:  b.State[coin].Date,
		lastsold:      b.State[coin].SaleDate,
		lastema:       b.State[coin].LastEma,
		lastsma:       b.State[coin].LastSma,
		purchaseprice: b.State[coin].PurchasePrice,
	}

	b.State[coin].LastEma = d.ema
	b.State[coin].LastSma = d.sma
	d.cooloffduration = cacheCooloffduration

	b.Now = time.Unix(b.MyTrainingData[pair][tick].Timestamp, 0) // convert time stamp to "now" time
	if d.coinbalance == 0 && d.lastsold.After(b.Now.Add(-d.cooloffduration)) {
		d.cooloffperiod = true
	}
	d.HeldForLongEnough = d.purchasedate.After(b.Now.Add(-time.Hour * 22)) // yuk

	return d
}

var cacheCooloffduration time.Duration
var cacheAnalysisFunc string
