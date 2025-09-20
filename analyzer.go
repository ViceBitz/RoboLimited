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

type SalesData struct {
	NumPoints          int     `json:"num_points"`
	Timestamp          []int64 `json:"timestamp"`
	AvgDailySalesPrice []int   `json:"avg_daily_sales_price"`
	SalesVolume        []int   `json:"sales_volume"`
}

type SalesPoint struct {
	Date               time.Time
	AvgDailySalesPrice int
	SalesVolume        int
}

// Calculates mean and SD of past sales data
func findMeanSD(id string) (float64, float64) {
	url := fmt.Sprintf(config.RolimonsSite, id)
	historyData, err := extractPriceSeries(url)

	log.Println(err)

	if err != nil {
		return 0, 0
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

	return mean, std
}

// Calculates Z-score of price relative to past sales data
func findZIndex(id string, price float64) float64 {
	mean, std := tools.SalesData[id].Mean, tools.SalesData[id].StdDev
	if mean == 0.0 && std == 0.0 {
		mean, std = findMeanSD(id)
	}
	z_score := (price - mean) / std

	fmt.Println("Z-Score: ", z_score, "| Mean: ", mean, "| SD: ", std)

	return z_score
}

// Analyzes historical time series data to determine if an item is price manipulated
func CheckProjected(id string, rap float64) bool {
	fmt.Println("Projected Check | ID:", id)
	z_score := findZIndex(id, rap)
	return z_score >= config.OutlierThreshold //z-score above certain threshold is outlier
}

// Uses Z-score of best price across past sales data to identify dip and make buy decision
func CheckDip(id string, bestPrice float64) bool {
	fmt.Println("Dip Check | ID:", id)
	z_score := findZIndex(id, bestPrice)
	return z_score <= config.DipThreshold //z-score below threshold is dip
}

func extractPriceSeries(url string) (*SalesData, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create chrome instance
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
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
	var salesData SalesData
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
		itemDetails := GetLimitedData()
		var sales_data []tools.StatsData
		var mu sync.Mutex
		var wg sync.WaitGroup

		//Initialize global sales data map to check which values needed
		tools.SalesData = tools.RetrieveSales()

		//Multithread scan Rolimon's for sales data
		maxThreads := 4
		semaphore := make(chan struct{}, maxThreads)

		for id, _ := range itemDetails.Items {
			wg.Add(1)
			go func(itemID string) {
				defer wg.Done()

				semaphore <- struct{}{}        // Acquire thread
				defer func() { <-semaphore }() // Release thread

				mean, SD := 0.0, 0.0
				if tools.SalesData[id].Mean != 0.0 {
					mean, SD = tools.SalesData[id].Mean, tools.SalesData[id].StdDev
				} else {
					mean, SD = findMeanSD(itemID)
				}

				//Check throttle to prevent excessive rate-limiting
				if mean == 0.0 && SD == 0.0 {
					time.Sleep(15 * time.Second)
				}

				mu.Lock()
				sales_data = append(sales_data, tools.StatsData{ID: itemID, Mean: mean, StdDev: SD})
				log.Println("(", len(sales_data), "/", len(itemDetails.Items), ")", "Reading sales data of", itemID, "| Mean:", mean, "| SD:", SD)
				mu.Unlock()
			}(id)
		}

		wg.Wait()
		log.Println("Caching into sales data file..")
		tools.StoreSales(sales_data)
	}
	//Initialize global sales data map
	tools.SalesData = tools.RetrieveSales()
}
