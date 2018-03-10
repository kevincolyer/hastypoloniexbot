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
	conf.SetDefault("Currency.target", "STR")
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
	flag.StringVar(&config, "config", "config", "config file to use")
	flag.BoolVar(&collectdata, "collectdata", false, "collect ticker data and save to data folder as unixtime.json")
	flag.BoolVar(&mergedata, "mergedata", false, "Merge all collected ticker data and save to data folder")
	flag.Parse()

	// load config file
	ConfInit(config)
	BotName = conf.GetString("BotControl.botname")
	LogInit(BotName + ".log")
	Info.Println("STARTING HastyPoloniexBot VERSION " + VERSION + " Bot name:" + BotName)

	defer store(state) // make sure state info is saved when program terminates

	if collectdata {
		Info.Println("Collecting ticker data")
		collectTickerData()
		return
	}
	if mergedata {
		Info.Println("Merging ticker data")
		mergeData()
		return
	}
	if conf.GetBool("BotControl.Active") == false {
		Info.Println("Active is FALSE - Quiting")
		return
	}
	if conf.GetBool("BotControl.Simulate") {
		Info.Println("Simulate Mode is ON")
	}
	if conf.GetBool("BotControl.SellSellSell") {
		Info.Println("SellSellSell detected - attemping to sell all held assets")
		//SellSellSell()
		// 		return
	}

	// all setup is done
	trade()

}
