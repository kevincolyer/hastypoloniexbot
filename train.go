package main

import (
	"encoding/csv"
	// 	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	// 	"sort"
	// 	"strconv"
	"gitlab.com/wmlph/poloniex-api"
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

	file := b.TrainingOutputDir + "/" + "training.csv" // with git commit ? with latest time ?
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

	// ------------------------------------------------------------
	// main loop
	// TODO use go routines to speed things on? Depends how slow!
	var tickerpairs []Pair
	for keys, _ := range b.MyTrainingData {
		tickerpairs = append(tickerpairs, keys) // b.Ticker has strings as keys
	}
	//fmt.Printf("tickerpairs %v\n", tickerpairs)

	b.Logging = false
	b.LogInfo("You wont see this!")

	// calc permutations
	lengthTrainingData := len(b.MyTrainingData["USDT_BTC"])
	// loop: over traing date using different analysis routines
	//  vary the state parameters
	//  loop: new state.
	permutations := lengthTrainingData * 28 * 20
	counter := 1
	for tb := 0.0010; tb < 0.02; tb += 0.0005 { //38
		b.Conf.Set("TradingRules.triggerbuy", tb)
		for ts := 0.0; ts < tb; ts += 0.0005 {
			b.Conf.Set("TradingRules.triggersell", ts)
			//             fmt.Println(b.Conf.Get("TradingRules.triggerbuy"),b.Conf.Get("TradingRules.triggersell"))
			//-----------------------------------------------------------
			b.NewState()
			b.Ticker = make(poloniex.Ticker)
			rand.Seed(1970) // a good year

			buys := 0
			sells := 0
			initialBalance := b.State["BTC"].Balance // profit or loss

			// loop training data
			for tick := 0; tick < lengthTrainingData; tick++ {
				counter++
				fmt.Printf("\rCalculating %v of %v ", Comma(float64(counter)), Comma(float64(permutations)))
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
				b, s := b.Trade()
				buys += b
				sells += s
				// prepare analyse?
				// call analyse not needed????

			}
			fmt.Println()

			// Get bitcoin value of all trades
			baseTotal := b.State["BTC"].Balance
			for _, coin := range targets {
				coinBalance := b.State[coin].Balance
				baseTotal += b.Ticker["BTC_"+coin].Last * coinBalance
			}

			// calc profit or loss
			fmt.Printf("tb %v ts %v buys %v sells %v profit %v\n", tb, ts, buys, sells, fc(baseTotal-initialBalance))
			r := strings.Split(fmt.Sprintf("%v %v %v %v %v", tb, ts, buys, sells, baseTotal-initialBalance), " ")

			//  stuff into CSV the profit, buys and sells for parameters varied and analysis chosen
			records = append(records, r)
		} // end ts
	} //end tb

	//----------------------------------------------------------
	// complete CSV
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
	d.cooloffduration, _ = time.ParseDuration(b.Conf.GetString("TradingRules.CoolOffDuration"))

	b.Now = time.Unix(b.MyTrainingData[pair][tick].Timestamp, 0) // convert time stamp to "now" time
	if d.coinbalance == 0 && d.lastsold.After(b.Now.Add(-d.cooloffduration)) {
		d.cooloffperiod = true
	}
	d.HeldForLongEnough = d.purchasedate.After(b.Now.Add(-time.Hour * 22)) // yuk

	return d
}
