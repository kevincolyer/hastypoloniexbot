package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	//	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"gitlab.com/wmlph/poloniex-api"
)

const VERSION = "0.1.3"

type coinstate struct {
	Coin           string
	Balance        float64
	Date           time.Time
	SaleDate       time.Time
	PurchasePrice  float64
	PurchaseAmount float64
	OrderNumber    string
	FiatValue      float64
	BaseValue      float64
	LastEma        float64
	LastSma        float64
	Misc           string
}

var (
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	conf     *viper.Viper
	exchange *poloniex.Poloniex
	state    map[string]*coinstate
	BotName  = "HastyPoloniexBot"
	ticker   poloniex.Ticker
	Logging  bool = false // initial state of logging

//         state *viper.Viper

)

type coinaction struct {
	coin    string
	action  int
	ranking float64
}

const (
	NOACTION = iota
	BUY
	SELL
)

const (
	LAST  = "_LAST_"
	TOTAL = "_TOTAL_"
	START = "_START_"
)

// CONF
func ConfInit(config string) {
	conf = viper.New()
	conf.SetConfigType("toml")
	conf.AddConfigPath(".")
	conf.SetConfigName(config) // name of config file (without extension)
	// set defaults here
	conf.SetDefault("Currency.target", "STR")
	conf.SetDefault("TradingRules.fragments", 1)
	conf.SetDefault("TradingRules.CoolOffDuration", "2h")
	//
	err := conf.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error reading config file: %s \n", err))
	}
}

// STATE
func StateInit() {
	state = make(map[string]*coinstate)
	statefile := conf.GetString("DataStore.filename")

	if _, err := os.Stat(statefile); os.IsNotExist(err) {
		// defaults
		state[conf.GetString("Currency.Base")] = &coinstate{Balance: 0.1, Coin: "BTC"}
		state[conf.GetString("Currency.Target")] = &coinstate{}
		state[LAST] = &coinstate{Coin: "BTC"}
		state[TOTAL] = &coinstate{Coin: "BTC"}
		storestate()
	} else {
		// load and unmarshal state file
		j, err := ioutil.ReadFile(statefile)
		if err != nil {
			panic(fmt.Errorf("Fatal error reading state file: %s \n", err))
		}
		if len(j) < 6 && string(j) == "null" {
			panic("Fatal error: state file is null. (Consider removing)")
		}
		err = json.Unmarshal(j, &state)
		if err != nil {
			panic(fmt.Errorf("Fatal error unmarshalling state file: %s \n", err))
		}
	}
}

func storestate() {
	// load and unmarshal state file
	j, err := json.Marshal(state)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for state file: %s \n", err))
	}
	err = ioutil.WriteFile(conf.GetString("DataStore.filename"), j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing state file: %s \n", err))
	}
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	// set up throttler so we don't make more than 6 api calls per sec.
	throttlerchan = make(chan tick)
	go throttler(throttlerchan)
}

func main() {
	var config string
	var collectdata bool
	var mergedata bool
	var preparedata bool
	var trainmode bool
	flag.StringVar(&config, "config", "config", "config file to use")
	flag.BoolVar(&collectdata, "collectdata", false, "collect ticker data and save to data folder as [unixtime].json")
	flag.BoolVar(&mergedata, "mergedata", false, "Merge all collected ticker data and save to data folder as data.json")
	flag.BoolVar(&preparedata, "preparedata", false, "Prepare collected ticker data for trianing runs")
	flag.BoolVar(&trainmode, "train", false, "Start a training run")
	flag.Parse()

	// load config file
	ConfInit(config)

	// config state
	StateInit()
	defer storestate() // make sure state info is saved when program terminates

	// initialise logging
	Logging = true
	BotName = conf.GetString("BotControl.botname")
	LogInit(BotName + ".log")
	if Logging {
		Info.Println("STARTING HastyPoloniexBot VERSION " + VERSION + " Bot name:" + BotName)
	}

	// Special runing modes
	if collectdata {
		if Logging {
			Info.Println("Collecting ticker data")
		}
		collectTickerData()
		return // end program
	}
	if mergedata {
		if Logging {
			Info.Println("Merging ticker data")
		}
		mergeData()
		return // end program
	}
	if preparedata {
		if Logging {
			Info.Println("Preparing ticker data")
		}
		prepareData()
		return // end program
	}
	if trainmode {
		if Logging {
			Info.Println("Entering training mode")
		}
		train()
		return // end program
	}
	if conf.GetBool("BotControl.Active") == false {
		if Logging {
			Info.Println("Active is FALSE - Quiting")
		}
		return // end program
	}

	// Modes of operation
	if conf.GetBool("BotControl.Simulate") {
		if Logging {
			Info.Println("Simulate Mode is ON")
		}
	}
	if conf.GetBool("BotControl.SellSellSell") {
		if Logging {
			Info.Println("SellSellSell detected - attemping to sell all held assets")
		}
	}

	// all setup is done
	trade()

}
