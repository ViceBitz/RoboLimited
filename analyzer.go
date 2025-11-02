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

	"github.com/chromedp/chromedp"
)

//Extracts past sales data and calculates mean and SD
func analyzeSales(id string) (float64, float64, *tools.Sales) {
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
		//Exclude points beyond lookback period
		if timestamps[len(timestamps)-1]-timestamps[i] > 24*60*60*config.LookbackPeriod {
			break
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
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev
	if mean == 0.0 && std == 0.0 {
		mean, std, _ = analyzeSales(id)
	}
	z_score := (price - mean) / std

	if logStats {
		fmt.Println("Z-Score: ", z_score, "| Mean: ", mean, "| SD: ", std)
	}

	return z_score
}

//Calculates coefficient of variation of past sales data
func findCV(id string) float64 {
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev
	if mean == 0.0 && std == 0.0 {
		mean, std, _ = analyzeSales(id)
	}
	cv := std / mean
	if config.LogConsole {
		fmt.Println("%CV : ", cv)
	}
	return cv
}

//Determine if an item is price manipulated with RAP z-score
func CheckProjected(id string, rap float64) bool {
	if (config.LogConsole) {
		fmt.Println("Projected Check | ID:", id)
	}
	z_score := findZScore(id, rap, config.LogConsole)
	return z_score >= config.OutlierThreshold //z-score above certain threshold is outlier
}

//Identify dip to support buy decision with price z-score
func CheckDip(id string, bestPrice float64) bool {
	if (config.LogConsole) {
		fmt.Println("Dip Check | ID:", id)
	}
	z_score := findZScore(id, bestPrice, config.LogConsole)
	cv := findCV(id)
	cutoff := -0.3/cv - config.DipThreshold //z-score below -0.3/%CV - threshold is dip
	if (config.LogConsole) {
		fmt.Println("Z-Score Cutoff: ", cutoff)
	}
	return z_score <= cutoff
}

//Calculate optimal price listing for item sale from z-score
func FindOptimalSell(id string) float64 {
	fmt.Println("Optimal Sale | ID:", id)
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev
	return mean + std*config.SellThreshold
}

//Scans for items with falling prices under z-score threshold, within price range, and at demand level
func SearchFallingItems(z_threshold float64, priceLow float64, priceHigh float64, isDemand bool) []string {
	itemDetails := tools.GetLimitedData()
	var fallingItems []string
	for id, _ := range itemDetails.Items {
		name := itemDetails.Items[id][0]
		rap := itemDetails.Items[id][2].(float64)
		demand := int(itemDetails.Items[id][5].(float64))
		price := rap
		z_score := findZScore(id, price, config.LogConsole)
		if z_score < z_threshold && priceLow <= price && price <= priceHigh && (!isDemand || demand != -1) {
			fallingItems = append(fallingItems, id)
			log.Println("Found item", name, "| ID:", id, "| Z-Score:", z_score)
		}
	}
	return fallingItems
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

	fmt.Printf("Extracted JSON data (%d characters)\n", len(salesDataJSON))

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
					mean, SD, historyData = analyzeSales(itemID)
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
