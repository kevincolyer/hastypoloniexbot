package main

import (
	"io"
	"log"
	"os"
)

func (b *Bot) LogInit(output string) {

	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", output, ":", err)
	}

	multi := io.MultiWriter(file, os.Stdout)

	// 	Info = log.New(multi,
	// 		"INFO: ",
	// 		log.Ldate|log.Ltime|log.Lshortfile)
	//
	// 	Warning = log.New(multi,
	// 		"WARNING: ",
	// 		log.Ldate|log.Ltime|log.Lshortfile)
	//
	// 	Error = log.New(multi,
	// 		"ERROR: ",
	// 		log.Ldate|log.Ltime|log.Lshortfile)

	b.BotLog = log.New(multi,
		"", 0)
}

// object based conditional logging.
func (b *Bot) LogError(s string) {
	if b.Logging {
		b.BotLog.Println("ERROR: " + s)
	}
}
func (b *Bot) LogErrorf(fmt string, args ...interface{}) {
	if b.Logging {
		b.BotLog.Printf("ERROR: "+fmt, args...)
	}
}

func (b *Bot) LogInfo(s string) {
	if b.Logging {
		b.BotLog.Println("INFO : " + s)
	}
}
func (b *Bot) LogInfof(fmt string, args ...interface{}) {
	if b.Logging {
		b.BotLog.Printf("INFO : "+fmt, args...)
	}
}

func (b *Bot) LogWarning(s string) {
	if b.Logging {
		b.BotLog.Println("WARN : " + s)
	}
}
func (b *Bot) LogWarningf(fmt string, args ...interface{}) {
	if b.Logging {
		b.BotLog.Printf("WARN : "+fmt, args...)
	}
}
