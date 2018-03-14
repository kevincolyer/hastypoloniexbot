package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
