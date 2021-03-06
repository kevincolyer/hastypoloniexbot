package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/spf13/viper"
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
	LastPrice      float64
	PurchaseAmount float64
	OrderNumber    string
	FiatValue      float64
	BaseValue      float64
	LastEma        float64
	LastSma        float64
	Misc           string
}

// var (
// 	Info    *log.Logger
// 	Warning *log.Logger
// 	Error   *log.Logger
// )

type Bot struct {
	Conf              *viper.Viper
	Exchange          *poloniex.Poloniex
	State             map[string]*coinstate
	Ticker            poloniex.Ticker
	BotLog            *log.Logger
	BotName           string
	Version           string
	Logging           bool
	Training          bool
	TrainingDataFile  string
	TrainingDataDir   string
	TrainingOutputDir string
	TrainingDataTick  int
	TrainingParams    string
	MyTrainingData    TrainingData
	Now               time.Time
}

func NewBot() *Bot {
	b := Bot{
		BotName:           BOTNAME,
		Version:           VERSION,
		Logging:           false, // initial state of logging
		Training:          false, // initial state of training
		TrainingDataDir:   "data",
		TrainingDataFile:  "trainingdata.json",
		TrainingOutputDir: "training",
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
	if a == NOACTION {
		return "NoAction"
	}
	if a == BUY {
		return "Buy"
	}
	if a == SELL {
		return "Sell"
	}
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
	if err != nil {              // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error reading config file: %s \n", err))
	}
}

// STATE
func (b *Bot) StateInit() {
	b.NewState()

	statefile := b.Conf.GetString("DataStore.filename")

	if _, err := os.Stat(statefile); os.IsNotExist(err) {
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

func (b *Bot) NewState() {
	b.State = make(map[string]*coinstate)
	// defaults
	b.State[b.Conf.GetString("Currency.Base")] = &coinstate{Balance: 1, Coin: "BTC"} // updated this and also minbasetotrade=0.2
	b.State[b.Conf.GetString("Currency.Target")] = &coinstate{}
	b.State[LAST] = &coinstate{Coin: "BTC"}
	b.State[TOTAL] = &coinstate{Coin: "BTC"}
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
	var preparedata bool
	var trainmode bool
	var traincoins string
	var trainparams string

	flag.StringVar(&config, "config", "config", "config file to use")
	flag.BoolVar(&collectdata, "collectdata", false, "collect ticker data and save to data folder as [unixtime].json")
	flag.BoolVar(&preparedata, "preparedata", false, "Prepare collected ticker data for trianing runs")
	flag.BoolVar(&trainmode, "train", false, "Start a training run")
	flag.StringVar(&traincoins, "traincoins", "all", "coins to train with: 'config'=use config file, 'all' or comma separated list 'ETH,STR' etc")
	flag.StringVar(&trainparams, "trainparams", "", "comma sep list of training params. 'lim=1000,tblo=0.008,af=02' af is analyse function. ")
	flag.Parse()

	// make Bot object
	var b = NewBot()
	// load config file
	b.ConfInit(config)

	// config state
	b.StateInit()
	defer b.StoreState() // make sure state info is saved when program terminates

	// initialise logging
	b.Logging = true
	b.BotName = b.Conf.GetString("BotControl.botname")
	b.LogInit(b.BotName + ".log")
	b.LogInfo(getTimeNowString() + " STARTING HastyPoloniexBot VERSION " + b.Version + " Bot name:" + b.BotName)
	defer b.LogInfo(getTimeNowString() + " FINISHED")
	// Special data collection/training modes

	// collect data: get ticker data to build training data from
	if collectdata {
		b.LogInfo("Collecting ticker data")
		b.CollectTickerData()
		return // end program
	}

	// preparedata: combine ticker date collected by collectdata with some processing and filtering
	if preparedata {
		b.LogInfo("Preparing ticker data")
		b.PrepareData()
		return // end program
	}

	// trainmode: fine tune params and analysis strategies using training data
	if trainmode {
		b.Training = true
		b.Conf.Set("BotControl.Simulate", "true") // really don't want to use real data!
		b.LogInfo("Entering training mode")
		b.TrainingParams = trainparams
		b.Train(traincoins)
		return // end program
	}

	// Trading modes

	// Bot config says Bot should not be active
	if b.Conf.GetBool("BotControl.Active") == false {
		b.LogInfo("Active is FALSE - Quiting")
		return // end program
	}

	// Simulate mode is set to on/true
	if b.Conf.GetBool("BotControl.Simulate") {
		b.LogInfo("Simulate Mode is ON")
	}

	// Crash sell is on/true
	if b.Conf.GetBool("BotControl.SellSellSell") {
		b.LogInfo("SellSellSell detected - attemping to sell all held assets")
	}

	// Setup is done
	b.Trade()
}
