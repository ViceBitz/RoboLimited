package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"robolimited/config"
	"robolimited/tools"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chewxy/stl"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

//Extracts price data, resamples to 1-day snapshots, calculates mean/SD within date range
func processPriceSeries(id string, daysLower int64, daysUpper int64) (float64, float64, *tools.Sales, []int) {
	url := fmt.Sprintf(config.RolimonsSite, id)
	historyData, err := extractPriceSeries(url)
	dayUnit := int64(24*60*60) //1 day in seconds

	var pricePoints []int
	if err != nil {
		return 0, 0, historyData, pricePoints
	}

	// Find segment of time series to consider
	pricePointsAll := historyData.AvgDailySalesPrice
	t := historyData.Timestamp
	var prevTimestamp int64
	var prevPrice int

	for i := len(pricePointsAll)-1; i>=0; i-- {
		//Only look at sales data within interval [today-daysLower, today-daysUpper]
		if t[len(t)-1]-t[i] > dayUnit*daysLower {
			break //Exclude points before (today - daysLower)
		}
		if t[len(t)-1]-t[i] < dayUnit*daysUpper {
			continue //Don't scan points after (today - daysUpper)
		}
		
		if (prevTimestamp != 0) {
			timeGap := int64(prevTimestamp) - t[i]
			//Append intermediary price points if prev_t - t[i] > 1 day
			if (timeGap > dayUnit) {
				priceDiff := prevPrice - pricePointsAll[i]
				slope := priceDiff / int(timeGap) //slope from i+1 -> i
				missingDays := int(timeGap / dayUnit)
				for k := 1; k <= missingDays; k++ {
					pricePoints = append(pricePoints, int(prevPrice + slope * k * int(dayUnit))) //assume linear
				}
				prevTimestamp -= int64(missingDays) * dayUnit
				prevPrice += slope * missingDays * int(dayUnit)
			}
			timeGap = prevTimestamp - t[i]
			//Skip points if prev_t - t[i] < 1 day
			if (timeGap < dayUnit) {
				prevTimestamp = t[i]
				prevPrice = pricePointsAll[i]
				continue
			}
		} else {
			pricePoints = append(pricePoints, pricePointsAll[i])
			prevTimestamp = t[i];
			prevPrice = pricePointsAll[i]
		}
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

	//Reverse price points to be chronological
	for i, j := 0, len(pricePoints)-1; i < j; i, j = i+1, j-1 {
        pricePoints[i], pricePoints[j] = pricePoints[j], pricePoints[i]
    }

	return mean, std, historyData, pricePoints
}

//Calculates Z-score of price relative to past sales data; pulls from cached data if exists
func findZScore(id string, price float64, logStats bool) float64 {
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev //Use cache for fast query
	if mean == 0.0 && std == 0.0 { //Scrape mean and SD if not cached
		mean, std, _, _ = processPriceSeries(id, config.LookbackPeriod, 0) //Get data from lookback period
	}
	z_score := (price - mean) / std

	if logStats {
		fmt.Println("Z-Score: ", z_score, "| Mean: ", mean, "| SD: ", std)
	}

	return z_score
}

/*
Calculates Z-score of seasonal spike across specified date range, and projects
future price using that dated z-score from past year.

Let L1 = today-daysL1, U1 = today-daysU1, L2 = today-daysL2, U2 = today-daysU2.
This computes the z-score using the price average of date range [L1, U1] as the target
and the distribution across date range [L2, U2] as the reference/baseline.

We predict a future price by casting last year's z-score change to this year, in the
following manner: P_future = P_avg_current + dated_z_score * sd_current
*/
func projectPrice_ZScore(id string, daysL1 int64, daysU1 int64, daysL2 int64, daysU2 int64, logStats bool) (float64, float64) {
	//**Does not read from sales data cache, use "findZScore" for that
	mean, sd, _, _ := processPriceSeries(id, daysL2, daysU2) //Reference distribution
	avgPrice, _, _, _ := processPriceSeries(id, daysL1, daysU1) //Target mean

	//Compute preceding mean's z-score across date range
	z_score := (avgPrice - mean) / sd
	if logStats {
		fmt.Println("Dated Z-Score: ", z_score, "| Ref. Mean: ", mean, "| Target Price: ", avgPrice)
	}
	//Predict future price using past year's trend
	curMean, curSD := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev //Use cache for fast query
	if curMean == 0.0 && curSD == 0.0 { //Scrape mean and SD if not cached
		curMean, curSD, _, _ = processPriceSeries(id, config.LookbackPeriod, 0) //Get data from lookback period
	}
	priceFuture := curMean + z_score * curSD

	return z_score, priceFuture
}

//Models 30-day future pattern from noisy data using Holt's linear trend method
func holtLinear(data []float64, alpha, beta float64, steps int) ([]float64, float64) {
	level := data[0]
	trend := data[1] - data[0]
	for t := 1; t < len(data); t++ {
		prevLevel := level
		level = alpha*data[t] + (1-alpha)*(level+trend)
		trend = beta*(level-prevLevel) + (1-beta)*trend
	}
	forecast := make([]float64, steps)
	for i := 0; i < steps; i++ {
		forecast[i] = level + float64(i+1)*trend
	}
	sum := 0.0
	for _, v := range forecast {
		sum += v
	}
	nextAvg := sum / float64(steps)
	return forecast, nextAvg
}

/*
Uses STL (season-trend) decomposition to forecast a price average for future date range.

We break down past prices as a long-term trend component (T), seasonal cyclical component (S),
and random residue (R). Once we have models for T and S, we can predict future prices with
T_avg(today + next) + S_avg(today + next)
*/
func projectPrice_STL(id string, daysBefore int64, daysFuture int64, logStats bool) (float64) {
	_, _, _, pricePoints := processPriceSeries(id, daysBefore, 0)

	//Prepare price series for decomposition
	priceSeries := make([]float64, len(pricePoints))
	for i, v := range pricePoints {
		priceSeries[i] = float64(v)
	}
	if (len(priceSeries) < 2) { //Too short, error out
		return math.NaN()
	}
	//Decompose price series into trend and seasonal components with STL
	period := 7 //weekly cycles
	width := 31
	res := stl.Decompose(priceSeries, period, width, stl.Additive())

	
	//Predict future 30 days with linear trend forecast
	trend := res.Trend
	seasonal := res.Seasonal

	n := len(trend)
    if n < 2 {
        return -1
    }

	//Future prediction with Holt's and P = T + S
	trendForecast, _ := holtLinear(trend, 0.8, 0.2, int(daysFuture))

	sum := 0.0
	for h := 1; h <= int(daysFuture); h++ {
		tFuture := trendForecast[h-1] //predicted trend from Holt's
		seasonalIdx := (n + h) % period //seasonal component
		sFuture := seasonal[seasonalIdx]
		yFuture := tFuture + sFuture //assume residue = 0
		sum += yFuture
	}

	if (logStats) {
		fmt.Println(n)
		//Plot time-series models
		p := plot.New()
		p.Title.Text = "STL Decomposition Models"
		p.Y.Label.Text = "Price"

		ptsPlot := make(plotter.XYs, n)
		trendPlot := make(plotter.XYs, n+int(daysFuture))
		seasonPlot := make(plotter.XYs, n)
		for i := 0; i < n; i++ {
			t := i
			trendPlot[i].X = float64(t)
			seasonPlot[i].X = float64(t)
			ptsPlot[i].X = float64(t)
			trendPlot[i].Y = trend[i]
			seasonPlot[i].Y = seasonal[i]
			ptsPlot[i].Y = priceSeries[i]
		}
		for i := 0; i < int(daysFuture); i++ {
			trendPlot[n+i].X = float64(n+i)
			trendPlot[n+i].Y = trendForecast[i]
		}
		lpTrend, _ := plotter.NewLine(trendPlot)
		lpSeason, _ := plotter.NewLine(seasonPlot)
		dots, _ := plotter.NewScatter(ptsPlot)

		lpTrend.Color = color.RGBA{R: 30, G: 144, B: 255, A: 255}
		lpSeason.Color = color.RGBA{R: 160, G: 32, B: 240, A: 255}
		dots.Color = color.Black

		p.Add(lpTrend, lpSeason, dots)
		p.Add(plotter.NewGrid())
		p.Save(800, 400, "data/stl_model.png")
	}

    return sum / float64(daysFuture)
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


type Item struct {
	id string
	z_score float64
}
//Scans dated z-scores of items compared to past means within price range, demand level, and date range
func SearchDatedWithin(z_low float64, z_high float64, priceLow float64, priceHigh float64, daysL1 int64, daysU1 int64, daysL2 int64, daysU2 int64, isDemand bool) []string {
	itemDetails := tools.GetLimitedData()
	var itemsWithin []Item
	for id, _ := range itemDetails.Items {
		name := itemDetails.Items[id][0]
		rap := itemDetails.Items[id][2].(float64)
		demand := int(itemDetails.Items[id][5].(float64))
		price := rap

		//Filter out items outside price range and demand
		if priceLow <= price && price <= priceHigh && (!isDemand || demand != -1) {
			z_score, priceFuture := projectPrice_ZScore(id, daysL1, daysU1, daysL2, daysU2, config.LogConsole)
			if z_low <= z_score && z_score <= z_high {
				itemsWithin = append(itemsWithin, Item{id, z_score})
			}
			fmt.Println("Processed item:", name, "| Z-Score:", z_score, "| Price Prediction:", priceFuture)
			time.Sleep(3 * time.Second) //Avoid rate-limiting
		}
		
	}
	//Sort by ascending z-score
	sort.Slice(itemsWithin, func(i, j int) bool {
		return itemsWithin[i].z_score < itemsWithin[j].z_score
	})
	var onlyItems []string
	for _, m := range itemsWithin {
		name := itemDetails.Items[m.id][0]
		rap := itemDetails.Items[m.id][2]
		onlyItems = append(onlyItems, m.id)
		fmt.Println("Found item", name, "| ID:", m.id, "| RAP:", rap, "| Z-Score:", m.z_score)
	}

	return onlyItems
}


//Scans z-scores of items within price range and demand level
func SearchItemsWithin(z_low float64, z_high float64, priceLow float64, priceHigh float64, isDemand bool) []string {
	itemDetails := tools.GetLimitedData()
	var itemsWithin []Item
	for id, _ := range itemDetails.Items {
		rap := itemDetails.Items[id][2].(float64)
		demand := int(itemDetails.Items[id][5].(float64))
		price := rap

		//Filter out items outside price range and demand
		if priceLow <= price && price <= priceHigh && (!isDemand || demand != -1) {
			z_score := findZScore(id, price, config.LogConsole)
			if z_low <= z_score && z_score <= z_high {
				itemsWithin = append(itemsWithin, Item{id, z_score})
			}
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

	//Use previous year's price trends to forecast future seasonal cycles
	if (forecastPrices) {
		var tot_past_z float64 //Total forecasted z-scores
		var weighted_past_z float64 //Weighted forecasted z-scores
		fmt.Println("Forecasts:")
		for _,id := range assetIds {
			if len(itemDetails.Items[id]) == 0 { continue }
			name := itemDetails.Items[id][0]
			rap := itemDetails.Items[id][2].(float64)
			//Examine z-score from 2 months last year compared to its preceding 30 days
			past_z_score, priceFuture := projectPrice_ZScore(id, 330, 270, 450, 360, config.LogConsole)
			if (past_z_score != past_z_score) {
				continue //Check for NaN
			}
			tot_past_z += past_z_score
			weighted_past_z += rap * past_z_score
			fmt.Println(name, "| Forecast Z-Score:", past_z_score, "| Price Prediction:", priceFuture)
		}
		fmt.Println()
		fmt.Println("Avg. Forecast Z-Score: ", (tot_past_z / float64(itemsProcessed)), " | ", "Weighted Forecast Z-Score: ", (weighted_past_z / float64(tot_rap)))
		fmt.Println("____________________________________________________")
	}
	fmt.Println("Listed Items: ", fmt.Sprintf("%d", itemsProcessed) + "/" + fmt.Sprintf("%d",len(assetIds)))
}

//Extracts time-series sales data from Rolimon's asset URL
func extractPriceSeries(url string) (*tools.Sales, error) {
	//Extract raw HTML from item page source
	resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch url: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %v", err)
    }
    html := string(body)

	//Find sales data embedded within source using regex search
	re := regexp.MustCompile(`var\s+sales_data\s*=\s*(\{[\s\S]*?\});`)
	match := re.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("sales data not found in page HTML")
	}
	salesDataJSON := strings.TrimSuffix(match[1], ";")

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
					mean, SD, historyData, _ = processPriceSeries(itemID, config.LookbackPeriod, 0)
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
