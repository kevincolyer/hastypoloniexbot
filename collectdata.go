package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/wmlph/poloniex-api"
)

func collectTickerData() {
	/////////////////////////////////////
	// get poloniex data and set up variables from config file

	//if Logging { Info.Printf("%v%v", conf.GetString("Credentials.apikey"),conf.GetString("Credentials.secret")) }
	exchange = poloniex.NewKS(
		conf.GetString("Credentials.apikey"),
		conf.GetString("Credentials.secret")) // check for failure needed here?

	// Get Ticker
	// {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
	ticker, err := exchange.Ticker()
	if err != nil {
		if Logging {
			Error.Printf("Fatal error getting ticker data from poloniex: %v", err)
		}
		return
	}

	// load and unmarshal state file
	j, err := json.Marshal(ticker)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for ticker data: %s \n", err))
	}
	file := "data/" + fmt.Sprintf("%d", time.Now().Unix()) + ".json"
	err = ioutil.WriteFile(file, j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing data file: %s \n", err))
	}
	if Logging {
		Info.Println(file + " written ok.")
	}
}

func mergeData() {
	files, err := ioutil.ReadDir("data")
	if err != nil {
		if Logging {
			Error.Printf("Fatal error reading data directory: %v (is it created?)", err)
		}
		return
	}

	merged := "{"
	for _, file := range files {
		fname := strings.Split(file.Name(), ".")
		filename := "data/" + file.Name()

		// filter the directory listing...
		if len(fname) != 2 || fname[1] != "json" || fname[0] == "data" {
			if Logging {
				Info.Print("skipping " + filename)
			}
			continue
		}

		// read the file data...
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			if Logging {
				Info.Print(fmt.Errorf("Fatal error reading file: (%s) %s ", filename, err))
			}
			continue
		}

		// filter data to remove
		// ...

		// if Logging { Info.Print("merging "+filename) }
		if len(merged) > 1 {
			merged += ",\n"
		}
		merged += fmt.Sprintf("\"%s\": %s", fname[0], data)
	}
	merged += "}" // "}\n"
	if merged == "{}" {
		if Logging {
			Info.Print("No data to merge. Not writing data/data.json")
		}
		return
	}

	err = ioutil.WriteFile("data/data.json", []byte(merged), 0664)
	if err != nil {
		if Logging {
			Error.Printf("Fatal error writing data file: %s ", err)
		}
	}
	if Logging {
		Info.Println("data.json written ok.")
	}
}

//,"XMR_NXT":{"Last":"0.00056971","lowestAsk":"0.00058588","highestBid":"0.00057017","percentChange":"-0.1225241","baseVolume":"34.89409024","quoteVolume":"57047.21168544","isFrozen":"0"}
//  ,"[A-Z]+_[A-Z]+":\{.+\}

/* TAKE COLLECTED DATA AND TRANSFORM INTO A DIFFERENT ORDERED FORM AND CALCULATE EMA SMA
 * STORE AS JSON TO BE USED FOR DRY RUNS
 */

// timestamp.json: ticker== map of pairs, map of prices --> add ema, sma and timestamp fields to prices [ticker with  extra fields]
//          transforms to
// map of pairs: slice of prices sorted by timestamp

type (
	pair string // "BTC_STR"

	TickerEntryPlus struct {
		Last        float64 `json:",string"`
		Ask         float64 `json:"lowestAsk,string"`
		Bid         float64 `json:"highestBid,string"`
		Change      float64 `json:"percentChange,string"`
		BaseVolume  float64 `json:"baseVolume,string"`
		QuoteVolume float64 `json:"quoteVolume,string"`
		IsFrozen    int64   `json:"isFrozen,string"`

		Ema30     float64 `json:"ema30,string"`
		Sma50     float64 `json:"sma50,string"`
		Timestamp int64   `json:"timestamp,string"` // TODO not expoterd!!! Capitalise to export...
	}

	TickerPlus map[pair]TickerEntryPlus

	TrainingData map[pair][]TickerEntryPlus
)

func prepareData() {
	// test to see if data has been collected
	// init data structures
	// call mergeData()
	// if success...
	//  open data file
	//  unmarshall
	//  stuff into new data structure
	//
	// for each pair:
	//  for each timestamp in order
	//   calc sma
	//   calc ema
	//  truncate EMA-SMA of data
	//
	// save

	myTrainingData := make(map[pair][]TickerEntryPlus)

	// open data directory
	files, err := ioutil.ReadDir("data")
	if err != nil {
		if Logging {
			Error.Printf("Fatal error reading data directory: %v (is it created?)", err)
		}
		return
	}

	// open each file
	for i, file := range files {
		fname := strings.Split(file.Name(), ".")
		filename := "data/" + file.Name()

		fmt.Printf("\rReading file %v/%v ", i+1, len(files))

		// filter the directory listing...
		timestamp, err := strconv.Atoi(fname[0])
		if len(fname) != 2 || fname[1] != "json" || err != nil {
			if Logging {
				Info.Print("skipping " + filename)
			}
			continue
		}

		// read the file data...
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			if Logging {
				Info.Print(fmt.Errorf("Fatal error reading file: (%s) %s ", filename, err))
			}
			continue
		}
		//fmt.Println(fname,len(data))

		myTickerPlus := make(map[pair]TickerEntryPlus)

		err = json.Unmarshal(data, &myTickerPlus)
		if err != nil {
			panic(fmt.Errorf("Fatal error unmarshalling data file: %s \n", err))
		}

		// remap data and apply some filtering...
		for myPair, myTickerEntry := range myTickerPlus {
			// filter out non BTC
			if !strings.HasPrefix(string(myPair), "BTC_") {
				continue
			}
			myTickerEntry.Timestamp = int64(timestamp)
			myTickerEntry.Ema30 = 30
			myTickerEntry.Sma50 = 50
			myTrainingData[myPair] = append(myTrainingData[myPair], myTickerEntry)
		}
	} // end for each file

	fmt.Println("Sorting...")
	// data is currently unsorted. sort on timestamp key here
	for myPair, _ := range myTrainingData {
		sort.Slice(myTrainingData[myPair], func(i, j int) bool { return myTrainingData[myPair][i].Timestamp < myTrainingData[myPair][j].Timestamp })
	}
	// do transforms on data
	fmt.Printf("\rPre-caching Moving Average %v   ", "BTC_STR") //myPair)
	fmt.Println()
	// TODO

	// marshall to json and save
	j, err := json.Marshal(myTrainingData)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for training data: %s \n", err))
	}
	file := "data/trainingdata.json"
	err = ioutil.WriteFile(file, j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing data file: %s \n", err))
	}
	if Logging {
		Info.Println(file + " written ok.")
	}

}

// move to own file...
func train() {
	return
}
