package main

import (
	"encoding/json"
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

const APPNAME = "HastyPoloniexBot"

type coinstate struct {
	Coin          string
	Balance       float64
	Date          time.Time
	PurchasePrice float64
	OrderNumber   string
}

var (
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	conf     *viper.Viper
	exchange *poloniex.Poloniex
	state    map[string]*coinstate

//         state *viper.Viper

)

const (
	NOACTION = iota
	BUY
	SELL
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

func ConfInit() {
	// CONF
	conf = viper.New()
	conf.SetConfigType("toml")
	conf.AddConfigPath(".")
	conf.SetConfigName("config") // name of config file (without extension)
	// set defaults here
	//
	err := conf.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error reading config file: %s \n", err))
	}
	Info.Println("Loaded config file")

	// STATE
	state = make(map[string]*coinstate)
	statefile := conf.GetString("DataStore.filename")
	if _, err := os.Stat(statefile); os.IsNotExist(err) {
		// defaults
		state[conf.GetString("Currency.Base")] = &coinstate{Balance: 0.1}
		state[conf.GetString("Currency.Target")] = &coinstate{}
		state["LAST"] = &coinstate{Coin: "BTC"}
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
	Info.Println("Loaded state information")

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
	Info.Println("Stored state information")
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	LogInit(APPNAME + ".log")
	Info.Println("STARTING " + APPNAME)
	// load config file
	ConfInit()
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
	// 	ticker, _ := exchange.Ticker()
	// 	fmt.Println(ticker)

	Info.Println("Analysing Data")
	action, ranking := Analyse(conf.GetString("Currency.Target"))
	//         fmt.Println(action,ranking)

	Info.Println("Cancelling all Open Orders")

	if action == BUY {
		if ranking >= 0 {
			// best ranking?
		}
		Info.Println("Placing BUY  order")
		Buy("BTC", "STR", 0.001, 3)
	}
	if action == SELL {

		Info.Println("Placing SELL order")
	}
	if action == NOACTION {

		Info.Println("Nothing to do")
	}
	////////////////////////////////////
}
