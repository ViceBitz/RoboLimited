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

type HistoryData struct {
	NumPoints  int     `json:"num_points"`
	Timestamp  []int64 `json:"timestamp"`
	Favorited  []int   `json:"favorited"`
	RAP        []int   `json:"rap"`
	BestPrice  []int   `json:"best_price"`
	NumSellers []int   `json:"num_sellers"`
}

type PricePoint struct {
	Date      time.Time
	RAP       int
	BestPrice int
	Favorited int
	Sellers   int
}

// Analyzes historical time series data to determine if an item is price manipulated
func CheckProjected(id string, rap float64) bool {
	url := fmt.Sprintf(config.RolimonsSite, id)
	historyData, err := extractPriceSeries(url)

	if err != nil {
		return true
	}

	// Calculate z-index of point
	pricePoints_all := convertToPoints(historyData)
	pricePoints := pricePoints_all[len(pricePoints_all)-config.ProjectedPriceHistory:]
	mean := 0.0
	for _, p := range pricePoints {
		mean += float64(p.RAP)
	}
	N := float64(len(pricePoints))
	mean /= N

	std := 0.0
	for _, p := range pricePoints {
		std += math.Pow((float64(p.RAP) - mean), 2)
		fmt.Println(p.RAP)
	}
	std = math.Sqrt(std / (N - 1))
	z_score := (rap - mean) / std
	fmt.Println("Projected Check | ID:", id, "| Z-Score", z_score)

	return z_score > 2
}

func extractPriceSeries(url string) (*HistoryData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	var historyDataJSON string

	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`
			(function() {
				if (typeof window.history_data !== 'undefined') {
					return JSON.stringify(window.history_data);
				} else {
					// Try alternative variable names
					var alternatives = ['history_data', 'historyData', 'priceHistory', 'itemHistory'];
					for (var i = 0; i < alternatives.length; i++) {
						var varName = alternatives[i];
						if (typeof window[varName] !== 'undefined') {
							return JSON.stringify(window[varName]);
						}
					}
					
					// If not found, return available window properties for debugging
					var windowProps = [];
					for (var prop in window) {
						if (prop.toLowerCase().includes('history') || prop.toLowerCase().includes('data') || prop.toLowerCase().includes('price')) {
							windowProps.push(prop);
						}
					}
					
					return JSON.stringify({
						error: "history_data not found",
						available_properties: windowProps,
						window_keys_count: Object.keys(window).length
					});
				}
			})()
		`, &historyDataJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("chrome automation failed: %v", err)
	}

	// Check if error
	var errorCheck map[string]interface{}
	if json.Unmarshal([]byte(historyDataJSON), &errorCheck) == nil {
		if errorMsg, exists := errorCheck["error"]; exists {
			fmt.Printf("Error from browser: %v\n", errorMsg)
			return nil, fmt.Errorf("history_data not available in browser")
		}
	}

	// Parse history data
	var historyData HistoryData
	err = json.Unmarshal([]byte(historyDataJSON), &historyData)
	if err != nil {
		// Show first part of JSON for debugging
		preview := historyDataJSON
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("Failed to parse history data JSON: %v\nJSON preview: %s", err, preview)
	}

	return &historyData, nil
}

func convertToPoints(data *HistoryData) []PricePoint {
	points := make([]PricePoint, len(data.RAP))

	for i := 0; i < len(data.RAP); i++ {
		points[i] = PricePoint{
			Date:      time.Unix(data.Timestamp[i], 0),
			RAP:       data.RAP[i],
			BestPrice: getBestPrice(data.BestPrice, i),
			Favorited: getFavorited(data.Favorited, i),
			Sellers:   getSellers(data.NumSellers, i),
		}
	}

	return points
}

func getBestPrice(arr []int, index int) int {
	if index < len(arr) {
		return arr[index]
	}
	return 0
}

func getFavorited(arr []int, index int) int {
	if index < len(arr) {
		return arr[index]
	}
	return 0
}

func getSellers(arr []int, index int) int {
	if index < len(arr) {
		return arr[index]
	}
	return 0
}
