package main

import (
	"fmt"
	"time"
        "encoding/json"
	"io/ioutil"
        "strings"
        
	"gitlab.com/wmlph/poloniex-api"
)

func collectTickerData() {
	/////////////////////////////////////
	// get poloniex data and set up variables from config file
	Info.Println("GETTING POLONIEX DATA")

	//Info.Printf("%v\n%v\n", conf.GetString("Credentials.apikey"),conf.GetString("Credentials.secret"))
	exchange = poloniex.NewKS(
		conf.GetString("Credentials.apikey"),
		conf.GetString("Credentials.secret")) // check for failure needed here?

	// Get Ticker
	// {Last, Ask, Bid,Change,BaseVolume,QuoteVolume,IsFrozen}
	ticker, err := exchange.Ticker()
	if err != nil {
		Error.Printf("Fatal error getting ticker data from poloniex: %v\n", err)
		return
	}

	
	// load and unmarshal state file
	j, err := json.Marshal(ticker)
	if err != nil {
		panic(fmt.Errorf("Fatal error marshalling json for ticker data: %s \n", err))
	}
	file:="data/"+fmt.Sprintf("%d", time.Now().Unix() )+".json"
	err = ioutil.WriteFile(file, j, 0664)
	if err != nil {
		panic(fmt.Errorf("Fatal error writing data file: %s \n", err))
	}
}

/* merge data
for all data files
    filename chop .json bit
    write into data.json file
    with { filename: DATA } \n
    loop
save
*/

func mergeData() {
        files, err := ioutil.ReadDir("data")
	if err != nil {
            Error.Printf("Fatal error reading data directory: %v (is it created?)\n", err)
            return
	}
	
	merged:="{"
	for _, file := range files {
                fname:=strings.Split(file.Name(),".")
                filename:="data/"+file.Name()
                if len(fname)!=2 || fname[1]!="json" || fname[0]=="data" { 
                    Info.Print("skipping "+filename)
                    continue 
                }
                
                data,err:=ioutil.ReadFile(filename)
                if err != nil {
                        Info.Print(fmt.Errorf("Fatal error reading file: (%s) %s \n", filename, err))
                        continue
		}
                
                // Info.Print("merging "+filename)
                if len(merged)>1 { merged+=",\n"}
                merged+=fmt.Sprintf("\"%s\": %s",fname[0],data)
        }
        merged+="}" // "}\n"
        if merged=="{}" {
            Info.Print("No data to merge. Not writing data/data.json")
            return
        }
        
        err = ioutil.WriteFile("data/data.json", []byte(merged), 0664)
	if err != nil {
		Error.Printf("Fatal error writing data file: %s \n", err)
	}
	Info.Println("data.json written ok.")
}
