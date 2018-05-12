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

func (b *Bot) CollectTickerData() {
	/////////////////////////////////////
	// get poloniex data and set up variables from config file

	//if b.Logging { Info.Printf("%v%v", b.Conf.GetString("Credentials.apikey"),b.Conf.GetString("Credentials.secret")) }
	b.Exchange = poloniex.NewKS(
		b.Conf.GetString("Credentials.apikey"),
		b.Conf.GetString("Credentials.secret")) // check for failure needed here?

	// Get Ticker
	// {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
	ticker, err := b.Exchange.Ticker()
	if err != nil {
		if b.Logging {
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
	if b.Logging {
		Info.Println(file + " written ok.")
	}
}

func (b *Bot) MergeData() {
	files, err := ioutil.ReadDir("data")
	if err != nil {
		if b.Logging {
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
			if b.Logging {
				Info.Print("skipping " + filename)
			}
			continue
		}

		// read the file data...
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			if b.Logging {
				Info.Print(fmt.Errorf("Fatal error reading file: (%s) %s ", filename, err))
			}
			continue
		}

		// filter data to remove
		// ...

		// if b.Logging { Info.Print("merging "+filename) }
		if len(merged) > 1 {
			merged += ",\n"
		}
		merged += fmt.Sprintf("\"%s\": %s", fname[0], data)
	}
	merged += "}" // "}\n"
	if merged == "{}" {
		if b.Logging {
			Info.Print("No data to merge. Not writing data/data.json")
		}
		return
	}

	err = ioutil.WriteFile("data/data.json", []byte(merged), 0664)
	if err != nil {
		if b.Logging {
			Error.Printf("Fatal error writing data file: %s ", err)
		}
	}
	if b.Logging {
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

	// NOTE: to make a part of the struct not export (it will not import) just change the initial letter to lowercase
	TickerEntryPlus struct {
		Last        float64 `json:",string"`
		Ask         float64 `json:"lowestAsk,string"`
		Bid         float64 `json:"highestBid,string"`
		change      float64 `json:"percentChange,string"` // not currently exported
		baseVolume  float64 `json:"baseVolume,string"`    // not currently exported
		quoteVolume float64 `json:"quoteVolume,string"`   // not currently exported
		IsFrozen    int64   `json:"isFrozen,string"`

		Ema30     float64 `json:"ema30,string"`
		Sma50     float64 `json:"sma50,string"`
		Timestamp int64   `json:"timestamp,string"` // TODO not expoterd!!! Capitalise to export...
	}

	TickerPlus map[pair]TickerEntryPlus

	TrainingData map[pair][]TickerEntryPlus
)

func (b *Bot) PrepareData() {
	// test to see if data has been collected
	// init data structures
	// unmarshall
	// stuff into new data structure
	//
	// for each pair:
	//  for each timestamp in order
	//   calc sma
	//   calc ema
	//  truncate EMA-SMA of data
	// save in
	datadir := "data"
	trainingdatafile := "trainingdata.json"

	myTrainingData := make(map[pair][]TickerEntryPlus)

	// open data directory
	files, err := ioutil.ReadDir(datadir)
	if err != nil {
		if b.Logging {
			Error.Printf("Fatal error reading data directory: %v (is it created?)", err)
		}
		return
	}
	if len(files) < 100 {
		panic("Not enough data files. Need at least 100!")
	}
	// open each file
	for i, file := range files {
		fname := strings.Split(file.Name(), ".")
		filename := datadir + "/" + file.Name()

		fmt.Printf("\rReading file %v/%v ", i+1, len(files))

		// filter the directory listing...
		timestamp, err := strconv.Atoi(fname[0])
		if len(fname) != 2 || fname[1] != "json" || err != nil {
			if b.Logging {
				Info.Print("skipping " + filename)
			}
			continue
		}

		// read the file data...
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			if b.Logging {
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

	//fmt.Println("Sorting...")
	// data is currently unsorted. sort on timestamp key here
	for myPair, _ := range myTrainingData {
		sort.Slice(myTrainingData[myPair], func(i, j int) bool { return myTrainingData[myPair][i].Timestamp < myTrainingData[myPair][j].Timestamp })

		// do transforms on data
		// aim to truncate at sma50 as that requires the most data and only accurate from that point
		trunc := 50
		fmt.Printf("\rPre-caching Moving Average %v   ", myPair)

		// sma on data   myTrainingData[myPair][i].Last etc
		var sum float64
		var emamulti float64 = 2.0 / (30.0 + 1)

                datalength := len(myTrainingData[myPair])
		if datalength < 100 {
			panic("Not enough data loaded to create moving averages")
		}

		for i := 0; i < trunc; i++ {
			sum += myTrainingData[myPair][i].Last
		}
		myTrainingData[myPair][trunc-1].Sma50 = sum / 50 // first SMA50
		for i := trunc; i < datalength; i++ {
			myTrainingData[myPair][i].Sma50 = myTrainingData[myPair][i-1].Sma50 + (myTrainingData[myPair][i].Last-myTrainingData[myPair][i-50].Last)/50
		}
		// ema on data see CalcEMA
		// get initial sma10
		var ema float64
		var sma10 float64
		for i := trunc - 30 - 10; i < trunc-30; i++ {
			sma10 += myTrainingData[myPair][i].Last
		}
		sma10 = sma10 / 10
		for i := trunc - 30; i < datalength-30; i++ {
			// rolling sma10
			sma10 = sma10 + (myTrainingData[myPair][i].Last-myTrainingData[myPair][i-10].Last)/10 // rolling SMA to start
			// rolling sma10 is starting point for ema30
			ema = sma10
			// roll forward 30
			for j := 0; j < 30; j++ {
				ema = (myTrainingData[myPair][i+j].Last-ema)*emamulti + ema
			}
			myTrainingData[myPair][i+30].Ema30 = ema

		}
		// truncate data
		myTrainingData[myPair] = myTrainingData[myPair][trunc:]

	}
	fmt.Println()

	// marshall to json and save
	j, err := json.Marshal(myTrainingData)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for training data: %s \n", err))
	}
	file := datadir + "/" + trainingdatafile //"data/trainingdata.json"
	err = ioutil.WriteFile(file, j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing data file: %s \n", err))
	}
	if b.Logging {
		Info.Println(file + " written ok.")
	}

}


func loadPreparedData() *TrainingData {
    // check it exists
    myTrainingData := make(TrainingData)
//     myTrainingData := make(map[pair][]TickerEntryPlus)
    // load and unmarshall
    // sort
    
    // return
	return &myTrainingData
}

// move to own file...
func (b *Bot) Train()  {

}
