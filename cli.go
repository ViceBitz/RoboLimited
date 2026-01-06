package main

import (
	"flag"
	"fmt"
	"log"
	"robolimited/config"
	"robolimited/tools"
	"strings"
)

/*
Command-line interface to run various modules and operations
*/

// Start deal sniper process
func monitor() {
	snipeDeals(config.LiveMoney)
}

// Displays player inventory metrics
func analyzeInventory(forecast_type string) {
	AnalyzeInventory(true, forecast_type)
}

// Assess future value of item trade
func analyzeTrade(giveItems []string, receiveItems []string, daysPast int64, daysFuture int64) {
	EvaluateTrade(giveItems, receiveItems, daysPast, daysFuture)
}

// Finds current price-lowering items in market
func searchDips(threshold float64, priceLow float64, priceHigh float64, isDemand bool) {
	SearchFallingItems(threshold, priceLow, priceHigh, isDemand)
}

// Forecast growth potential with z-score analysis
func searchForecast(priceLow float64, priceHigh float64, daysPast int64, daysFuture int64, isDemand bool, sortBy string) {
	ForecastWithin(-1000, 1000, priceLow, priceHigh, daysPast, daysFuture, isDemand, sortBy)
}

// Scan for item owners within net worth range
func searchOwners(itemId string, worth_low float64, worth_high float64, limit int) {
	FindOwners(itemId, worth_low, worth_high, limit)
}

// General forecaster
func forecast(forecastItems []string, daysPast int64, daysFuture int64) {
	itemDetails := tools.GetLimitedData()
	for _, id := range forecastItems {
		name := itemDetails.Items[id][0]
		rap := itemDetails.Items[id][2].(float64)

		log.Println("____________________________________________________")
		//Forecast prices with z-score analysis
		z_score, priceFuture := modelZScore(id, 330, 270, 450, 360, false)
		log.Println(name, "(Z-Score) | Z-Score:", z_score, "| Price Prediction:", priceFuture)

		//Forecast prices with STL decomposition
		priceSTL, stability, peaks, dips, p_ratios, d_ratios := modelFourierSTL(id, daysPast, daysFuture, true)
		z_score_stl := findZScoreRelativeTo(id, priceSTL, rap, false)
		log.Println(name, "(STL) | Z-Score:", z_score_stl, "| RAP:", rap, "| Price Prediction:", priceSTL)
		log.Println("Stability (Resid. %CV):", stability)
		log.Println("Peaks:", peaks)
		log.Println("Dips:", dips)
		log.Println("Peak Ratios:", p_ratios)
		log.Println("Dip Ratios:", d_ratios)
	}
}

func main() {
	// Define the main mode flag
	mode := flag.String("mode", "", "Which function to run: monitor, analyzeInventory, analyzeTrade, searchDips, searchForecast, forecast, executor")

	// Flags for analyzeTrade
	give := flag.String("give", "", "Comma-separated list of items to give")
	receive := flag.String("receive", "", "Comma-separated list of items to receive")

	// Flags for analyzeInventory
	forecastType := flag.String("forecast_type", "stl", "Forecast type for inventory analysis")

	// Flags for searches
	threshold := flag.Float64("threshold", -0.5, "Threshold for price dips")
	priceLow := flag.Float64("priceLow", 0.0, "Minimum price for search")
	priceHigh := flag.Float64("priceHigh", 1000000.0, "Maximum price for search")
	isDemand := flag.Bool("isDemand", true, "Only include high-demand items")
	itemId := flag.String("item", "", "Specific item to target")
	limit := flag.Int("limit", 20, "Max records to output")
	sortBy := flag.String("sortBy", "z-score", "Attribute to order items by")

	// Flags for forecast
	items := flag.String("items", "", "Comma-separated list of items to forecast")
	daysPast := flag.Int64("daysPast", 365*5, "Number of past days of historical data to include in the forecast")
	daysFuture := flag.Int64("daysFuture", 30, "Number of days forward to project avg. price")

	flag.Parse()

	switch *mode {
	case "monitor":
		monitor()

	case "analyzeInventory":
		analyzeInventory(*forecastType)

	case "analyzeTrade":
		if *give == "" || *receive == "" {
			fmt.Println("Please provide both -give and -receive items for analyzeTrade")
			return
		}
		giveItems := strings.Split(*give, ",")
		receiveItems := strings.Split(*receive, ",")
		analyzeTrade(giveItems, receiveItems, *daysPast, *daysFuture)

	case "searchDips":
		searchDips(*threshold, *priceLow, *priceHigh, *isDemand)

	case "searchForecast":
		searchForecast(*priceLow, *priceHigh, *daysPast, *daysFuture, *isDemand, *sortBy)

	case "searchOwners":
		if *itemId == "" {
			fmt.Println("Please provide a target item id")
			return
		}
		searchOwners(*itemId, *priceLow, *priceHigh, *limit)

	case "forecast":
		if *items == "" {
			fmt.Println("Please provide -items for forecast")
			return
		}
		forecastItems := strings.Split(*items, ",")
		forecast(forecastItems, *daysPast, *daysFuture)

	default:
		fmt.Println("Unknown mode:", *mode)
	}
}
