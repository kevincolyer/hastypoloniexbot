package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"gitlab.com/wmlph/poloniex-api"
)

type coinstate struct {
	Coin           string
	Balance        float64
	Date           time.Time
	PurchasePrice  float64
	PurchaseAmount float64
	OrderNumber    string
	FiatValue      float64
	BaseValue      float64
}

var (
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	conf     *viper.Viper
	exchange *poloniex.Poloniex
	state    map[string]*coinstate
	BotName  = "HastyPoloniexBot"

//         state *viper.Viper

)

const (
	NOACTION = iota
	BUY
	SELL
)

const (
	LAST  = "_LAST_"
	TOTAL = "_TOTAL_"
)

func LogInit(output string) {

	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", output, ":", err)
	}

	multi := io.MultiWriter(file, os.Stdout)
	Info = log.New(multi,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(multi,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(multi,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func ConfInit(config string) {
	// CONF
	conf = viper.New()
	conf.SetConfigType("toml")
	conf.AddConfigPath(".")
	conf.SetConfigName(config) // name of config file (without extension)
	// set defaults here
	//
	err := conf.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error reading config file: %s \n", err))
	}

	// STATE
	state = make(map[string]*coinstate)
	statefile := conf.GetString("DataStore.filename")
	if _, err := os.Stat(statefile); os.IsNotExist(err) {
		// defaults
		state[conf.GetString("Currency.Base")] = &coinstate{Balance: 0.1, Coin: "BTC"}
		state[conf.GetString("Currency.Target")] = &coinstate{}
		state[LAST] = &coinstate{Coin: "BTC"}
		state[TOTAL] = &coinstate{Coin: "BTC"}
		store(state)
	} else {

		// load and unmarshal state file
		j, err := ioutil.ReadFile(statefile)
		if err != nil {
			panic(fmt.Errorf("Fatal error reading state file: %s \n", err))
		}
		err = json.Unmarshal(j, &state)
		if err != nil {
			panic(fmt.Errorf("Fatal error unmarshalling state file: %s \n", err))
		}
	}
	// all ok

}

func store(s map[string]*coinstate) {
	// load and unmarshal state file
	j, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for state file: %s \n", err))
	}
	err = ioutil.WriteFile(conf.GetString("DataStore.filename"), j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing state file: %s \n", err))
	}
	// 	Info.Println("Stored state information")
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type coinaction struct {
	coin    string
	action  int
	ranking float64
}

func main() {
	var config string
	flag.StringVar(&config, "config", "config", "config file to use")
	flag.Parse()

	ConfInit(config)
	BotName = conf.GetString("BotControl.botname")
	LogInit(BotName + ".log")
	Info.Println("STARTING " + BotName)
	//       	Info.Println("Loaded config file")
	//	Info.Println("Loaded state information")

	// load config file
	defer store(state) // make sure state info is saved when program terminates

	if conf.GetBool("BotControl.Active") == false {
		Info.Println("Active is FALSE - Quiting")
		return
	}
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulate Mode is ON")
	}
	if conf.GetBool("BotControl.SellSellSell") {
		Info.Println("SellSellSell detected - attemping to sell all held assets")
		SellSellSell()
		return
	}

	// Note private api limited to 5 calls per second - perhaps have a sync chanel here that causes a pause if >  5 per sec?

	/////////////////////////////////////
	Info.Println("Getting Poloniex data")

	exchange = poloniex.NewKS("blah", "blah")

	// Ticker
	ticker, err := exchange.Ticker()
	if err != nil {
		Error.Printf("Fatal error getting ticker data from poloniex: %v\n", err)
		return
	}
	// {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
	fiat := conf.GetString("Currency.Fiat")
	base := conf.GetString("Currency.Base")
	basebalance := state[base].Balance
	FIATBTC := ticker[fiat+"_BTC"].Last // can be other curency than usdt
	state[base].FiatValue = basebalance * FIATBTC
	state[base].BaseValue = basebalance // for completeness
	Info.Printf("BALANCE %v %v (%v %v) \n", fc(basebalance), base, fc(basebalance*FIATBTC), fiat)

	coin := conf.GetString("Currency.Target")
	//////////////////////////////////////
	// MULTICOIN VARIANT
	targets := strings.Split(conf.GetString("Currency.targets"), ",")
	if len(targets) == 0 {
		targets = append(targets, coin)
	}
	var coinbalance float64
	basetotal := basebalance
	var fragmenttotal float64

	for _, coin = range targets {
		if _, ok := state[coin]; !ok {
			state[coin] = &coinstate{Coin: coin, Balance: 0.0}

		}
		coinbalance = state[coin].Balance
		infiat := ticker[base+"_"+coin].Last * coinbalance * FIATBTC
		inbase := ticker[base+"_"+coin].Last * coinbalance

		Info.Printf("BALANCE %v %v (%v %v) \n", fc(coinbalance), coin, fc(infiat), fiat)
		if coinbalance > 0 {
			fragmenttotal++
			basetotal += inbase
		}
		state[coin].FiatValue = infiat
		state[coin].BaseValue = inbase
	}
	Info.Printf("BALANCE Total %v %v over %v coins", fc(basetotal), base, fragmenttotal)
	state[TOTAL].Balance = basetotal
	state[TOTAL].FiatValue = basetotal * FIATBTC

	////////////////////////////////////////////
	// Analyse coins
	Info.Println("Analysing Data")
	var todo []coinaction

	for _, coin = range targets {
		action, ranking := Analyse(coin)
		todo = append(todo, coinaction{coin: coin, action: action, ranking: ranking})
	}
	// sort by ranking descending
	sort.Slice(todo, func(i, j int) bool { return todo[i].ranking > todo[j].ranking })

	///////////////////////////////////////////
	Info.Println("Cancelling all Open Orders")
	// TODO

	///////////////////////////////////////////
	// buying and selling for each coin
	minbasetotrade := conf.GetFloat64("TradingRules.minbasetotrade")
	maxfragments := conf.GetFloat64("TradingRules.fragments")
	if maxfragments == 0 {
		maxfragments = 1
	}

	for i, _ := range todo {
		coin = todo[i].coin
		coinbalance := state[coin].Balance
		action := todo[i].action
		basebalance := state[base].Balance
		if action == BUY && coinbalance > 0 {
			Info.Printf(coin+" BUY cannot proceed as already hold %v\n", fc(coinbalance))
		}
		if action == BUY && coinbalance == 0 {
			// check enough balance to make an order (minorder)
			// get current asking price

			if basebalance > minbasetotrade {
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
			Info.Println(coin + " Placing SELL order")
			Sell(base, coin, ticker[base+"_"+coin].Bid, coinbalance)

		}
		if action == NOACTION {

			Info.Print(coin + " Nothing to do")
		}
	}

	//	Sell(base,"STR",ticker["BTC_STR"].Bid, state["STR"].Balance)
	////////////////////////////////////
	//update state before saving
	basetotal = state[base].Balance
	state[base].FiatValue = basetotal * FIATBTC
	state[base].BaseValue = basetotal
	s := fmt.Sprintf("coin|balance|BTC|Fiat\n")
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
	////////////////////////////////////
}
