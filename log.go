package main

import (
	"io"
	"log"
	"os"
)

var Logging bool = false // initial state of logging

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

func LogError (s string, i ...interface{}) {
    if Logging { Error.Printf(s,i) }
    return
}

func LogInfo (s string, i ...interface{}) {
    if Logging { Info.Printf(s,i) }
    return
}

func LogWarning (s string, i ...interface{}) {
    if Logging { Warning.Printf(s,i) }
    return
}
