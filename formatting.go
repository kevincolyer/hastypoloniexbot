package main

import (
	"fmt"
	"strings"
)

// format as percentage *100 make int
func fp(p float64) (s string) {
	s = fmt.Sprintf("%v", int(p*100))
	return
}

// format as percentage *100 make int 2 decimal places
func fp2(p float64) (s string) {
	s = fmt.Sprintf("%.2f", p*100)
	return
}

// format number
func fn(a float64) (s string) {
	s = fmt.Sprintf("%.8f", a)
	return
}

// as above but for 2 decimal places
func fn2(a float64) (s string) {
	s = fmt.Sprintf("%.2f", a)
	return
}

// FORMAT Currency
func fc(c float64) (s string) {
	i := strings.Split(fmt.Sprintf("%.9f", c), ".")
	s = everyThird(reverseStr(i[0]), ",")
	s = reverseStr(s) + "." + everyThird(i[1], " ")
	return
}

func Comma(n float64) string {
	i := strings.Split(fmt.Sprintf("%.2f", n), ".")
	return reverseStr(everyThird(reverseStr(i[0]), ",")) + "." + i[1]
}

func everyThird(str, insert string) (s string) {
	if str == "" {
		return
	}
	for len(str) > 0 {
		l := len(str)
		if l > 3 {
			if str[3] == '-' || str[3] == '+' {
				l = 4
			} else {
				l = 3
			}
		}
		s = s + str[:l]
		str = str[l:]
		if len(str) > 0 {
			s += insert
		}
	}
	return
}

func reverseStr(str string) (out string) {
	for _, s := range str {
		out = string(s) + out
	}
	return
}

const poloniexTime = "2006-01-02 15:04:05"

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

// type CurrencyPair struct {
// 	Base  string
// 	Trade string
// }
//
// func NewCurrencyPair(s string) CurrencyPair {
// 	i := strings.Split(s, "_")
// 	var p CurrencyPair
// 	p.Base = i[0]
// 	if len(i) > 1 {
// 		p.Trade = i[1]
// 	}
// 	return p
// }
//
// func (p CurrencyPair) String() string {
// 	return fmt.Sprintf("%-4s/%4s", p.Trade, p.Base)
// }
//
// func (p CurrencyPair) Poloniex() string {
// 	return fmt.Sprintf("%s_%s", p.Base, p.Trade)
// }
