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

const VERSION = "0.2.0"
const BOTNAME = "HastyPoloniexBot"

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
//	conf     *viper.Viper
//	exchange *poloniex.Poloniex
//	state    map[string]*coinstate
//	BotName  = "HastyPoloniexBot"
//	ticker   poloniex.Ticker
//	Logging  bool = false // initial state of logging

//         state *viper.Viper

)

type Bot struct {
    Conf     *viper.Viper
	Exchange *poloniex.Poloniex
	State    map[string]*coinstate
	Ticker   poloniex.Ticker
	BotName  string
	Version  string
	Logging  bool 
}

func NewBot() *Bot  {
    b:=Bot{
        BotName: BOTNAME, 
        Version: VERSION,
        Logging: false,             // initial state of logging
    }
    return &b
}

type coinaction struct {
	Coin    string
	Action  action
	Ranking float64
}

type action int

const (
	NOACTION action = iota
	BUY
	SELL
)

func (a action) String() string {
    if a==NOACTION { return "NoAction" }
    if a==BUY { return "Buy" }
    if a==SELL { return "Sell" }
    return "Err unknown"
}

const (
	LAST  = "_LAST_"
	TOTAL = "_TOTAL_"
	START = "_START_"
)

// CONF
func (b *Bot) ConfInit(config string) {
	b.Conf = viper.New()

    // set defaults here
	b.Conf.SetConfigType("toml")
	b.Conf.AddConfigPath(".")
	b.Conf.SetConfigName(config) // name of config file (without extension)
	b.Conf.SetDefault("Currency.target", "STR")
	b.Conf.SetDefault("TradingRules.fragments", 1)
	b.Conf.SetDefault("TradingRules.CoolOffDuration", "2h")
	//
	err := b.Conf.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error reading config file: %s \n", err))
	}
}

// STATE
func (b *Bot) StateInit() {
	b.State = make(map[string]*coinstate)
	
    statefile := b.Conf.GetString("DataStore.filename")

	if _, err := os.Stat(statefile); os.IsNotExist(err) {
		// defaults
		b.State[b.Conf.GetString("Currency.Base")] = &coinstate{Balance: 0.1, Coin: "BTC"}
		b.State[b.Conf.GetString("Currency.Target")] = &coinstate{}
		b.State[LAST] = &coinstate{Coin: "BTC"}
		b.State[TOTAL] = &coinstate{Coin: "BTC"}
		b.StoreState()
	} else {
		// load and unmarshal state file
		j, err := ioutil.ReadFile(statefile)
		if err != nil {
			panic(fmt.Errorf("Fatal error reading state file: %s \n", err))
		}
		if len(j) < 6 && string(j) == "null" {
			panic("Fatal error: state file is null. (Consider removing)")
		}
		err = json.Unmarshal(j, &b.State)
		if err != nil {
			panic(fmt.Errorf("Fatal error unmarshalling state file: %s \n", err))
		}
	}
}

func (b *Bot) StoreState() {
	// load and unmarshal state file
	j, err := json.Marshal(&b.State)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for state file: %s \n", err))
	}
	err = ioutil.WriteFile(b.Conf.GetString("DataStore.filename"), j, 0664)
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

	var bot = NewBot()
	// load config file
	bot.ConfInit(config)

	// config state
	bot.StateInit()
	defer bot.StoreState() // make sure state info is saved when program terminates

	// initialise logging
	bot.Logging = true
	bot.BotName = bot.Conf.GetString("BotControl.botname")
	LogInit(bot.BotName + ".log")
	if bot.Logging {
		Info.Println("STARTING HastyPoloniexBot VERSION " + bot.Version + " Bot name:" + bot.BotName)
	}

	// Special runing modes
	if collectdata {
		if bot.Logging {
			Info.Println("Collecting ticker data")
		}
		bot.CollectTickerData()
		return // end program
	}
	if mergedata {
		if bot.Logging {
			Info.Println("Merging ticker data")
		}
		bot.MergeData()
		return // end program
	}
	if preparedata {
		if bot.Logging {
			Info.Println("Preparing ticker data")
		}
		bot.PrepareData()
		return // end program
	}
	if trainmode {
		if bot.Logging {
			Info.Println("Entering training mode")
		}
		bot.Train()
		return // end program
	}
	if bot.Conf.GetBool("BotControl.Active") == false {
		if bot.Logging {
			Info.Println("Active is FALSE - Quiting")
		}
		return // end program
	}

	// Modes of operation
	if bot.Conf.GetBool("BotControl.Simulate") {
		if bot.Logging {
			Info.Println("Simulate Mode is ON")
		}
	}
	if bot.Conf.GetBool("BotControl.SellSellSell") {
		if bot.Logging {
			Info.Println("SellSellSell detected - attemping to sell all held assets")
		}
	}

	// all setup is done
	bot.Trade()

}
