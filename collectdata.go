package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/wmlph/poloniex-api"
)

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
		b.LogErrorf("Fatal error getting ticker data from poloniex: %v", err)
		return
	}

	// load and unmarshal state file
	j, err := json.Marshal(ticker)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for ticker data: %s \n", err))
	}
	file := b.TrainingDataDir + "/" + fmt.Sprintf("%d", time.Now().Unix()) + ".json"
	err = ioutil.WriteFile(file, j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing data file: %s \n", err))
	}
	b.LogInfo(file + " written ok.")
}

func (b *Bot) PrepareData() {
	myTrainingData := make(map[pair][]TickerEntryPlus)

	// open data directory
	files, err := ioutil.ReadDir(b.TrainingDataDir)
	if err != nil {
		b.LogErrorf("Fatal error reading data directory: %v (is it created?)", err)
		return
	}
	if len(files) < 100 {
		panic("Not enough data files. Need at least 100!")
	}

	// open each file
	for i, file := range files {
		fname := strings.Split(file.Name(), ".")
		filename := b.TrainingDataDir + "/" + file.Name()

		fmt.Printf("\rReading file %v/%v ", i+1, len(files))

		// filter the directory listing...
		timestamp, err := strconv.Atoi(fname[0])
		if len(fname) != 2 || fname[1] != "json" || err != nil {
			b.LogInfo("skipping " + filename)
			continue
		}

		// read the file data...
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			b.LogInfof("Fatal error reading file: (%s) %s ", filename, err)
			continue
		}

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
	file := b.TrainingDataDir + "/" + b.TrainingDataFile //"data/trainingdata.json"
	err = ioutil.WriteFile(file, j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing data file: %s \n", err))
	}
	b.LogInfo(file + " written ok.")
}

func (b *Bot) loadPreparedData() *TrainingData {
	// check it exists
	myTrainingData := make(TrainingData)
	file := b.TrainingDataDir + "/" + b.TrainingDataFile
	// load and unmarshall

	// read the file data...
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(fmt.Errorf("Fatal error reading file: (%s) %s ", file, err))
	}

	err = json.Unmarshal(data, &myTrainingData)
	if err != nil {
		panic(fmt.Errorf("Fatal error unmarshalling data file: %s \n", err))
	}

	// sort
	for myPair, _ := range myTrainingData {
		sort.Slice(myTrainingData[myPair], func(i, j int) bool { return myTrainingData[myPair][i].Timestamp < myTrainingData[myPair][j].Timestamp })
	}

	return &myTrainingData
}

// move to own file... ???
func (b *Bot) Train() {

	b.LogInfo("Loading training data")
	myTrainingData := b.loadPreparedData()

	// open a csv file for dumping data
	// open data directory
	_, err := ioutil.ReadDir(b.TrainingDataDir)
	if err != nil {
		b.LogErrorf("Fatal error reading data directory: %v (is it created?)", err)
		return
	}

	file := b.TrainingOutputDir + "/" + "training.csv" // with git commit ? with latest time ?
	f, err := os.Create(file)
	if err != nil {
		panic(fmt.Errorf("Fatal error opening file for writing: %s \n", err))
	}
	defer f.Close()

	w := csv.NewWriter(f)
	records := [][]string{
		{"header0", "header1", "header2"},
	}

	// ------------------------------------------------------------
	// main loop
	fmt.Println(len((*myTrainingData)))
	// calc permutations
	// loop: over traing date using different analysis routines
	//  vary the state parameters
	//  loop: new state.
	b.NewState()
	b.Logging = false
	b.LogInfo("You wont see this!")
	rand.Seed(1970) // a good year
	//     loop: coins (all or range?)
	//      sum permutations and display for impatient analysers
	//      prepare analyse
	//      call analyse
	//      do buy and sell
	records = append(records, []string{"data0", "data1", "data2"})
	//  stuff into CSV the profit, buys and sells for parameters varied and analysis chosen
	// finish and rest!
	// TODO use go routines to speed things on? Depends how slow!

	// complete CSV
	b.Logging = true
	b.LogInfo("Writing results file: " + file)
	w.WriteAll(records) // calls Flush internally
	if err := w.Error(); err != nil {
		panic(fmt.Errorf("error writing csv:", err))
	}
}
