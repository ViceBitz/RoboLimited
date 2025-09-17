package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"robolimited/config"
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

// Analyzes historical time series data to determine if an item is price manipulated
func CheckProjected(id string, rap float64) bool {
	url := fmt.Sprintf(config.RolimonsSite, id)
	historyData, err := extractPriceSeries(url)

	if err != nil {
		return true
	}

	// Find segment of time series to consider
	pricePointsAll := historyData.AvgDailySalesPrice
	timestamps := historyData.Timestamp
	var pricePoints []int
	for i := len(timestamps) - 1; i >= 0; i-- {
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
		fmt.Println(p)
	}
	std = math.Sqrt(std / (N - 1))
	z_score := (rap - mean) / std
	fmt.Println("Projected Check | ID:", id)
	fmt.Println("Z-Score: ", z_score, "| Mean: ", mean, "| SD: ", std)

	return z_score > 2 //z-index > 2 (standard deviations) is an outlier
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
		// Navigate to the page
		chromedp.Navigate(url),

		// Wait for page to load
		chromedp.WaitReady("body"),

		// Wait a bit more for JavaScript to execute
		chromedp.Sleep(3*time.Second),

		// Execute JavaScript to get window.sales_data
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

func convertToPoints(data *SalesData) []SalesPoint {
	points := make([]SalesPoint, len(data.AvgDailySalesPrice))

	for i := 0; i < len(data.AvgDailySalesPrice); i++ {
		points[i] = SalesPoint{
			Date:               time.Unix(data.Timestamp[i], 0),
			AvgDailySalesPrice: data.AvgDailySalesPrice[i],
			SalesVolume:        getSafeInt(data.SalesVolume, i),
		}
	}

	return points
}

func getSafeInt(arr []int, index int) int {
	if index < len(arr) {
		return arr[index]
	}
	return 0
}
