package tools

/*
Simulates buy & sell actions and tracks portfolio virtually.
Enables testing without using live money
*/

import (
	"robolimited/config"
	"strconv"
)

type TradeSimulator struct {
	RobuxSpent  int
	RobuxGained int
	Portfolio   map[string][]int
}

// Constructor
func NewTradeSimulator() *TradeSimulator {
	return &TradeSimulator{
		RobuxSpent:  0,
		RobuxGained: 0,
		Portfolio:   make(map[string][]int),
	}
}

// Buy an item
func (ts *TradeSimulator) BuyItem(id string, name string, price int) {
	WriteLineToFile(config.ActionLogFile, "Bought "+name+" for "+strconv.Itoa(price))
	ts.Portfolio[id] = append(ts.Portfolio[id], price)
	ts.RobuxSpent += price
}

/* DEPRECATED
// Sell an item
func (ts *TradeSimulator) SellItem(id string, name string, price int, value int) {
	if value != -1 {
		ts.RobuxGained += int(float64(price) * 0.7)
		WriteLineToFile(config.ActionLogFile, "Sold "+name+" for "+strconv.Itoa(int(float64(price)*0.7)))
	} else {
		ts.RobuxGained += int(float64(value) * 0.7)
		WriteLineToFile(config.ActionLogFile, "Sold "+name+" for "+strconv.Itoa(int(float64(value)*0.7)))
	}

	newCosts := []int{}
	deleted := false
	for _, p := range ts.Portfolio[id] {
		if p == price && !deleted {
			deleted = true
			continue
		}
		newCosts = append(newCosts, p)
	}

	ts.Portfolio[id] = newCosts
}
*/

// Get item portfolio
func (ts *TradeSimulator) GetPortfolio() map[string][]int {
	return ts.Portfolio
}
