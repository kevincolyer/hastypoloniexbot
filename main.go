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
        "flag"

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
}

var (
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	conf     *viper.Viper
	exchange *poloniex.Poloniex
	state    map[string]*coinstate
        BotName = "HastyPoloniexBot"

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
		state[conf.GetString("Currency.Base")] = &coinstate{Balance: 0.1,Coin:"BTC"}
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
        var config string
        flag.StringVar(&config , "config", "config", "config file to use")
        flag.Parse()
    
	ConfInit(config)
	BotName=conf.GetString("BotControl.botname")
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
	if err!=nil {
            Error.Printf("Fatal error getting ticker data from poloniex: %v\n",err)
            return
        }
        // {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
        fiat:=conf.GetString("Currency.Fiat")
	coin:=conf.GetString("Currency.Target")
        base:=conf.GetString("Currency.Base")
        FIATBTC := ticker[fiat+"_BTC"].Last // can be other curency than usdt
        
        // use state if simulating, otherwise use real poloniex balance
        basebalance:=state[base].Balance
        coinbalance:=state[coin].Balance
        /*if conf.GetBool("BotControl.Simulate") {
        }*/
        // else {
        
//         // Balances
//         balances, err := exchange.BalancesAll()
// 	if err != nil {
// 		Error.Printf("Failed to get poloniex balances: %v\n",err))
// 		return
// 	}
//}
	// get and summarise all balances - update state
// 	total:=0.0
        
        Info.Printf("BALANCE %v %v (%v %v) \n",fc(basebalance),base,fc(basebalance*FIATBTC),fiat)
        Info.Printf("BALANCE %v %v (%v %v) \n",fc(coinbalance),coin,fc(ticker[base+"_"+coin].Last*coinbalance*FIATBTC),fiat)
	
	// Analyse coins
	Info.Println("Analysing Data")
	action, ranking := Analyse(coin)
	//         fmt.Println(action,ranking)

	Info.Println("Cancelling all Open Orders")

	if action == BUY {
		if ranking >= 0 {
			// best ranking?
		}
		// check enough balance to make an order (minorder)
		// get current asking price
                minbalance:=conf.GetFloat64("TradingRules.minbasetotrade")
                if basebalance>minbalance {
                    Info.Println("Placing BUY  order")
                    Buy(base, coin, ticker[base+"_"+coin].Ask,basebalance)
                    
                } else {
                    Info.Printf("Balance of %v is lower (%v) than minbasetotrade rule (%v) Can't place buy order\n",base,fc(basebalance),fc(minbalance))
                }
	}
// 	Info.Println("Placing FORCED BUY  order")
//         Buy(base, coin, ticker[base+"_"+coin].Ask,basebalance)
                    
	if action == SELL {
                // get current bidding price
                // get balance and sell all
		Info.Println("Placing SELL order")
                Sell(base, coin,  ticker[base+"_"+coin].Bid,coinbalance)
                
            
        }
	if action == NOACTION {

		Info.Println("Nothing to do")
	}
	////////////////////////////////////
}
