package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"robolimited/config"
	"robolimited/tools"
	"sync"
	"time"
	"sort"

	"github.com/chromedp/chromedp"
)

//Extracts past sales data and calculates mean/SD within date range
func analyzeSales(id string, daysLower int64, daysUpper int64) (float64, float64, *tools.Sales) {
	url := fmt.Sprintf(config.RolimonsSite, id)
	historyData, err := extractPriceSeries(url)

	if err != nil {
		return 0, 0, historyData
	}

	// Find segment of time series to consider
	pricePointsAll := historyData.AvgDailySalesPrice
	timestamps := historyData.Timestamp
	var pricePoints []int
	for i := len(pricePointsAll) - 1; i >= 0; i-- {
		//Only look at sales data within interval [today-daysLower, today-daysUpper]
		if timestamps[len(timestamps)-1]-timestamps[i] > 24*60*60*daysLower {
			break //Exclude points before (today - daysLower)
		}
		if timestamps[len(timestamps)-1]-timestamps[i] < 24*60*60*daysUpper {
			continue //Don't scan points after (today - daysUpper)
		}
		pricePoints = append(pricePoints, pricePointsAll[i])
	}

	// Calculate z-index of point (across past points)
	mean := 0.0
	for _, p := range pricePoints {
		mean += float64(p)
	}
	N := float64(len(pricePoints))
	mean /= N

	std := 0.0
	for _, p := range pricePoints {
		std += math.Pow((float64(p) - mean), 2)
	}
	std = math.Sqrt(std / (N - 1))

	return mean, std, historyData
}

//Calculates Z-score of price relative to past sales data
func findZScore(id string, price float64, logStats bool) float64 {
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev //Use cache for fast query
	if mean == 0.0 && std == 0.0 { //Scrape mean and SD if not cached
		mean, std, _ = analyzeSales(id, config.LookbackPeriod, 0) //Get data from lookback period
	}
	z_score := (price - mean) / std

	if logStats {
		fmt.Println("Z-Score: ", z_score, "| Mean: ", mean, "| SD: ", std)
	}

	return z_score
}

//Calculates Z-score of price relative to past sales data within date range [today-daysLower, today-daysUpper]
func findDatedZScore(id string, price float64, daysLower int64, daysUpper int64, logStats bool) float64 {
	//**Does not read from sales data cache, use "findZScore" for that
	mean, std, _ := analyzeSales(id, daysLower, daysUpper) //Get data from date range
	z_score := (price - mean) / std
	if logStats {
		fmt.Println("Dated Z-Score: ", z_score, "| Mean: ", mean, "| SD: ", std)
	}
	return z_score
}

//Identify dip to support buy decision with price z-score
func CheckDip(id string, bestPrice float64, value float64, isDemand bool) bool {
	if (config.LogConsole) {
		fmt.Println("Dip Check | ID:", id)
	}
	
	//Different thresholds depending on item demand type
	threshold := config.DipThresholdND
	if isDemand {
		threshold = config.DipThresholdD
	}

	//Calculate z-score diff in comparison to break-even score
	z_score := findZScore(id, bestPrice, config.LogConsole)
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev

	worth := mean //Extrinsic value of item (avg. price or value)
	if value != -1 { worth = value }

	margin := config.MarginND //Discount margin below worth
	if isDemand { margin = config.MarginD }

	cutoff := (worth * (1 - margin) - mean)/std - threshold //z-score below break-even pt

	if (config.LogConsole) {
		fmt.Println("Z-Score Cutoff: ", cutoff)
	}
	//Margin cutoff + upper bound to protect against price manipulation
	return z_score <= cutoff && z_score <= config.DipUpperBound
}

//Scans z-scores of items within price range, demand level, and date range
func SearchItemsWithin(z_low float64, z_high float64, priceLow float64, priceHigh float64, isDemand bool) []string {
	type Item struct {
		id string
		z_score float64
	}

	itemDetails := tools.GetLimitedData()
	var itemsWithin []Item
	for id, _ := range itemDetails.Items {
		rap := itemDetails.Items[id][2].(float64)
		demand := int(itemDetails.Items[id][5].(float64))
		price := rap

		z_score := findZScore(id, price, config.LogConsole)
		if z_low <= z_score && z_score <= z_high && priceLow <= price && price <= priceHigh && (!isDemand || demand != -1) {
			itemsWithin = append(itemsWithin, Item{id, z_score})
		}
	}
	//Sort by ascending z-score
	sort.Slice(itemsWithin, func(i, j int) bool {
		return itemsWithin[i].z_score < itemsWithin[j].z_score
	})
	var onlyItems []string
	for _, m := range itemsWithin {
		name := itemDetails.Items[m.id][0]
		onlyItems = append(onlyItems, m.id)
		fmt.Println("Found item", name, "| ID:", m.id, "| Z-Score:", m.z_score)
	}

	return onlyItems
}
//Scans items under z-score threshold within price range and demand level in lookback period
func SearchFallingItems(z_high float64, priceLow float64, priceHigh float64, isDemand bool) []string {
	return SearchItemsWithin(-9999, z_high, priceLow, priceHigh, isDemand)
}

//Analyzes the z-scores of inventory items and prints list of metrics
func AnalyzeInventory(forecastPrices bool) {
	assetIds := tools.GetInventory(fmt.Sprintf("%d", config.RobloxId))
	itemDetails := tools.GetLimitedData()
	var tot_z float64 //Total z-score
	var weighted_z float64 //Weighted z-score
	var tot_rap float64 //Total RAP
	var itemsProcessed int //# of items successfully processed
	fmt.Println("____________________________________________________")
	for _, id := range assetIds {
		if len(itemDetails.Items[id]) == 0 { continue }
		name := itemDetails.Items[id][0]
		rap := itemDetails.Items[id][2].(float64)
		z_score := findZScore(id, rap, config.LogConsole)
		fmt.Println(name, "| Z-Score:", z_score)
		tot_z += z_score
		weighted_z += rap * z_score
		itemsProcessed += 1
		tot_rap += rap
	}
	fmt.Println()
	fmt.Println("Avg. Z-Score: ", (tot_z / float64(itemsProcessed)), " | ", "Weighted Z-Score: ", (weighted_z / float64(tot_rap)))
	fmt.Println("____________________________________________________")

	//Use previous year's prices to forecast future ones -> seasonal cycles
	if (forecastPrices) {
		var tot_past_z float64 //Total forecasted z-scores
		var weighted_past_z float64 //Weighted forecasted z-scores
		fmt.Println("Forecasts:")
		for _,id := range assetIds {
			if len(itemDetails.Items[id]) == 0 { continue }
			name := itemDetails.Items[id][0]
			rap := itemDetails.Items[id][2].(float64)
			//Look at z-score from 2 months exactly last year
			past_z_score := -findDatedZScore(id, rap, 360, 300, config.LogConsole) //Negative because current -> future is eq. to -(future -> current z-score))
			tot_past_z += past_z_score
			weighted_past_z += rap * past_z_score
			fmt.Println(name, "| Forecast Z-Score:", past_z_score)
		}
		fmt.Println()
		fmt.Println("Avg. Forecast Z-Score: ", (tot_past_z / float64(itemsProcessed)), " | ", "Weighted Forecast Z-Score: ", (weighted_past_z / float64(tot_rap)))
		fmt.Println("____________________________________________________")
	}
	fmt.Println("Listed Items: ", fmt.Sprintf("%d", itemsProcessed) + "/" + fmt.Sprintf("%d",len(assetIds)))
}

//Extracts time-series sales data from Rolimon's asset URL
func extractPriceSeries(url string) (*tools.Sales, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create chrome instance
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(config.UserAgent),
	)

	allocCtx, cancel2 := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel2()

	chromeCtx, cancel3 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel3()

	var salesDataJSON string

	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.Sleep(1*time.Second),

		chromedp.Evaluate(`
			(function() {
				return JSON.stringify(window.sales_data);
			})()
		`, &salesDataJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("chrome automation failed: %v", err)
	}

	if (config.LogConsole) {
		fmt.Printf("Extracted JSON data (%d characters)\n", len(salesDataJSON))
	}

	// Parse the actual sales data
	var salesData tools.Sales
	err = json.Unmarshal([]byte(salesDataJSON), &salesData)
	if err != nil {
		// Show first part of JSON for debugging
		preview := salesDataJSON
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("failed to parse sales data JSON: %v\nJSON preview: %s", err, preview)
	}

	return &salesData, nil
}

func init() {
	//Precompute mean & standard deviation for past sales data of all items
	//Write to a .csv file to use for querying later
	if config.PopulateSalesData {
		itemDetails := tools.GetLimitedData()

		var sales_stats []tools.StatsPoint
		sales_data := make(map[string]*tools.Sales)

		var mu sync.Mutex
		var wg sync.WaitGroup

		//Initialize global sales maps to check which values needed
		tools.SalesStats = tools.RetrieveSalesStats()
		tools.SalesData = tools.RetrieveSalesData()

		//Multithread scan Rolimon's for sales data
		maxThreads := 4
		semaphore := make(chan struct{}, maxThreads)

		for id, _ := range itemDetails.Items {
			wg.Add(1)
			go func(itemID string) {
				defer wg.Done()

				semaphore <- struct{}{}        // Acquire thread
				defer func() { <-semaphore }() // Release thread

				var historyData *tools.Sales

				mean, SD := 0.0, 0.0
				if tools.SalesStats[id].Mean != 0.0 {
					mean, SD = tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev
					historyData = tools.SalesData[id];
				} else {
					//Get data from lookback period
					mean, SD, historyData = analyzeSales(itemID, config.LookbackPeriod, 0)
				}

				//Check throttle to prevent excessive rate-limiting
				if mean == 0.0 && SD == 0.0 {
					time.Sleep(15 * time.Second)
				}

				mu.Lock()

				sales_stats = append(sales_stats, tools.StatsPoint{ID: itemID, Mean: mean, StdDev: SD})
				log.Println("(", len(sales_stats), "/", len(itemDetails.Items), ")", "Reading sales stats of", itemID, "| Mean:", mean, "| SD:", SD)
				
				if (historyData != nil) {
					sales_data[itemID] = historyData
					log.Println("Reading sales data. Length: ", len(historyData.AvgDailySalesPrice))
				}

				mu.Unlock()
			}(id)
		}

		wg.Wait()
		log.Println("Caching stats and data into files..")
		tools.StoreSalesStats(sales_stats)
		tools.StoreSalesData(sales_data)
	}
	//Initialize global sales maps
	tools.SalesStats = tools.RetrieveSalesStats()
	tools.SalesData = tools.RetrieveSalesData()
}
