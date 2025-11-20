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

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

/**
Toolkit/API for performing statistical data analysis on historical price day.
Drives decision-making and guides buying/selling/trading. Used across different
components in this project, both automatically and manually.
*/


// Extracts price data, resamples to 1-day snapshots, calculates mean/SD within date range
func processPriceSeries(id string, daysLower int64, daysUpper int64) (float64, float64, *tools.Sales, []int) {
	url := fmt.Sprintf(config.RolimonsSite, id)
	historyData, err := extractPriceSeries(url)
	dayUnit := int64(24 * 60 * 60) //1 day in seconds

	var pricePoints []int
	if err != nil {
		return 0, 0, historyData, pricePoints
	}

	// Find segment of time series to consider
	pricePointsAll := historyData.AvgDailySalesPrice
	t := historyData.Timestamp
	var prevTimestamp int64
	var prevPrice int

	for i := len(pricePointsAll) - 1; i >= 0; i-- {
		//Only look at sales data within interval [today-daysLower, today-daysUpper]
		if t[len(t)-1]-t[i] > dayUnit*daysLower {
			break //Exclude points before (today - daysLower)
		}
		if t[len(t)-1]-t[i] < dayUnit*daysUpper {
			continue //Don't scan points after (today - daysUpper)
		}

		if prevTimestamp != 0 {
			timeGap := int64(prevTimestamp) - t[i]
			//Append intermediary price points if prev_t - t[i] > 1 day
			if timeGap > dayUnit {
				priceDiff := prevPrice - pricePointsAll[i]
				slope := priceDiff / int(timeGap) //slope from i+1 -> i
				missingDays := int(timeGap / dayUnit)
				for k := 1; k <= missingDays; k++ {
					pricePoints = append(pricePoints, int(prevPrice+slope*k*int(dayUnit))) //assume linear
				}
				prevTimestamp -= int64(missingDays) * dayUnit
				prevPrice += slope * missingDays * int(dayUnit)
			}
			timeGap = prevTimestamp - t[i]
			//Skip points if prev_t - t[i] < 1 day
			if timeGap < dayUnit {
				prevTimestamp = t[i]
				prevPrice = pricePointsAll[i]
				continue
			}
		} else {
			pricePoints = append(pricePoints, pricePointsAll[i])
			prevTimestamp = t[i]
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

// Calculates Z-score of price relative to past sales data; pulls from cached data if exists
func findZScore(id string, price float64, logStats bool) float64 {
	mean, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev //Use cache for fast query
	if mean == 0.0 && std == 0.0 {                                      //Scrape mean and SD if not cached
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
	mean, sd, _, _ := processPriceSeries(id, daysL2, daysU2)    //Reference distribution
	avgPrice, _, _, _ := processPriceSeries(id, daysL1, daysU1) //Target mean

	//Compute preceding mean's z-score across date range
	z_score := (avgPrice - mean) / sd
	if logStats {
		fmt.Println("Dated Z-Score: ", z_score, "| Ref. Mean: ", mean, "| Target Price: ", avgPrice)
	}
	//Predict future price using past year's trend
	curMean, curSD := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev //Use cache for fast query
	if curMean == 0.0 && curSD == 0.0 {                                      //Scrape mean and SD if not cached
		curMean, curSD, _, _ = processPriceSeries(id, config.LookbackPeriod, 0) //Get data from lookback period
	}
	priceFuture := curMean + z_score*curSD

	return z_score, priceFuture
}

/*
Uses STL (season-trend) decomposition with Fourier regression to forecast
a price average for future date range.

We break down past prices as a long-term trend component (T), seasonal cyclical component (S).
T(t) is fitted with standard linear regression to account for price drift over time.
S(t) is modeled using Fourier regression to identify weekly and yearly cycles.

The model would take on the form:
y(t) = beta_0 + beta_1 * linearTrend(t) + fourierFeatures(t)
				   ^ T                  +        ^S

Once we have models for T and S, we can predict future prices by simplying extending the
model to (t, t + 1 ... t + future).
*/
func projectPrice_FourierSTL(id string, daysBefore int64, daysFuture int64, logStats bool) float64 {
	_, _, _, pricePoints := processPriceSeries(id, daysBefore, 0)

	//Prepare price series for decomposition
	priceSeries := make([]float64, len(pricePoints))
	for i, v := range pricePoints {
		priceSeries[i] = float64(v)
	}
	n := len(priceSeries)

	//Fourier regression + linear drift to model seasonality
	Kw := 3 //weekly order
	Ky := 5 //yearly order
	useYearly := n >= 400
	baseCols := 2
	p := baseCols + 2*Kw + 1
	if useYearly {
		p += 2 * Ky
	}

	X := make([][]float64, n)
	y := make([]float64, n)

	for t := 0; t < n; t++ {
		row := make([]float64, 0, p)
		row = append(row, 1.0)
		row = append(row, float64(t)/float64(n))

		//Weekly Fourier
		row = append(row, tools.FourierFeatures(t, 7.0, Kw)...)

		//Yearly Fourier
		if useYearly {
			row = append(row, tools.FourierFeatures(t, 365.25, Ky)...)
		}

		X[t] = row
		y[t] = priceSeries[t]
	}

	//Solve the regression for optimal betas
	beta, _ := tools.SolveNormalEq(X, y) 

	//Forecast prices in future days
	if daysFuture <= 0 {
		return math.NaN()
	}
	horizon := int(daysFuture)
	var sumF float64
	for h := 1; h <= horizon; h++ {
		t := n - 1 + h
		row := make([]float64, 0, p)
		row = append(row, 1.0)

		//Linear component
		row = append(row, float64(t)/float64(n))

		//Fourier periodic movements
		row = append(row, tools.FourierFeatures(t, 7.0, Kw)...)
		if useYearly {
			row = append(row, tools.FourierFeatures(t, 365.25, Ky)...)
		}

		var pred float64
		for j := 0; j < len(beta); j++ {
			pred += row[j] * beta[j]
		}
		if pred < 0 {
			pred = 0
		}
		sumF += pred
	}

	priceFuture := sumF / float64(horizon)

	if logStats {
		//Visualize Fourier regression
		fitted := make(plotter.XYs, n)
		weeklySeason := make(plotter.XYs, n)
		var yearlySeason plotter.XYs
		if useYearly {
			yearlySeason = make(plotter.XYs, n)
		}
		rawPts := make(plotter.XYs, n)

		idx := 0
		b0 := beta[idx]
		idx++
		bTrend := beta[idx]
		idx++
		
		//Construct linear trendline
		trendLine := make(plotter.XYs, n)
		for t := 0; t < n; t++ {
			x := float64(t)
			trendVal := b0 + bTrend * (float64(t) / float64(n))

			trendLine[t].X = x
			trendLine[t].Y = trendVal
		}

		//Construct Fourier seasonal component
		bWeekly := beta[idx : idx+2*Kw]
		idx += 2 * Kw
		var bYearly []float64
		if useYearly {
			bYearly = beta[idx : idx+2*Ky]
		}
		dot := func(a, b []float64) float64 {
			s := 0.0
			for i := 0; i < len(a); i++ {
				s += a[i] * b[i]
			}
			return s
		}

		for t := 0; t < n; t++ {
			x := float64(t)
			rawPts[t].X = x
			rawPts[t].Y = priceSeries[t]

			wf := tools.FourierFeatures(t, 7.0, Kw)
			weeklySeason[t].X = x
			weeklySeason[t].Y = dot(wf, bWeekly)

			ycomp := 0.0
			if useYearly {
				yf := tools.FourierFeatures(t, 365.25, Ky)
				yearlySeason[t].X = x
				yearlySeason[t].Y = dot(yf, bYearly)
				ycomp = yearlySeason[t].Y
			}

			trendVal := bTrend * (float64(t) / float64(n))
			fitted[t].X = x
			fitted[t].Y = b0 + trendVal + weeklySeason[t].Y + ycomp
			if fitted[t].Y < 0 {
				fitted[t].Y = 0
			}
		}

		var proj plotter.XYs
		if daysFuture > 0 {
			fh := int(daysFuture)
			proj = make(plotter.XYs, fh)
			for h := 1; h <= fh; h++ {
				t := n - 1 + h
				x := float64(t)
				wf := tools.FourierFeatures(t, 7.0, Kw)
				wsum := dot(wf, bWeekly)
				ysum := 0.0
				if useYearly {
					yf := tools.FourierFeatures(t, 365.25, Ky)
					ysum = dot(yf, bYearly)
				}
				trendVal := bTrend * (float64(t) / float64(n))
				yhat := b0 + trendVal + wsum + ysum
				if yhat < 0 {
					yhat = 0
				}
				proj[h-1].X = x
				proj[h-1].Y = yhat
			}
		}

		plt := plot.New()
		plt.Title.Text = "Fourier Seasonality: Fit & Projection"
		plt.X.Label.Text = "Day"
		plt.Y.Label.Text = "Price"

		dots, _ := plotter.NewScatter(rawPts)
		dots.Color = color.Black
		dots.Radius = vg.Points(1.2)

		lf, _ := plotter.NewLine(fitted)
		lf.Color = color.RGBA{R: 34, G: 139, B: 34, A: 255}
		lf.Width = vg.Points(2)

		lw, _ := plotter.NewLine(weeklySeason)
		lw.Color = color.RGBA{R: 30, G: 144, B: 255, A: 255}
		lw.Dashes = []vg.Length{vg.Points(5), vg.Points(3)}
		lw.Width = vg.Points(1.5)

		lt, _ := plotter.NewLine(trendLine)
		lt.Color = color.RGBA{R: 220, G: 20, B: 60, A: 255}
		lt.Width = vg.Points(2)

		plt.Add(lt)
		plt.Legend.Add("Trendline", lt)

		plt.Add(dots, lf, lw)
		plt.Legend.Add("Raw", dots)
		plt.Legend.Add("Fitted", lf)
		plt.Legend.Add("Weekly season", lw)

		if useYearly {
			ly, _ := plotter.NewLine(yearlySeason)
			ly.Color = color.RGBA{R: 160, G: 32, B: 240, A: 255}
			ly.Dashes = []vg.Length{vg.Points(3), vg.Points(3)}
			ly.Width = vg.Points(1.5)
			plt.Add(ly)
			plt.Legend.Add("Yearly season", ly)
		}
		if err := plt.Save(1000, 450, "data/fourier_stl_model.png"); err != nil {
			fmt.Println("plot save png:", err)
		}
	}

	return priceFuture
}

// Identify dip to support buy decision with price z-score
func CheckDip(id string, bestPrice float64, value float64, isDemand bool) bool {
	if config.LogConsole {
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
	if value != -1 {
		worth = value
	}

	margin := config.MarginND //Discount margin below worth
	if isDemand {
		margin = config.MarginD
	}

	cutoff := (worth*(1-margin)-mean)/std - threshold //z-score below break-even pt

	if config.LogConsole {
		fmt.Println("Z-Score Cutoff: ", cutoff)
	}
	//Margin cutoff + upper bound to protect against price manipulation
	return z_score <= cutoff && z_score <= config.DipUpperBound
}

type Item struct {
	id      string
	z_score float64
}

// Searches for items of STL price forecast within price range, demand level, and date range
func ForecastWithin(z_low float64, z_high float64, priceLow float64, priceHigh float64, daysPast int64, daysFuture int64, isDemand bool) []string {
	itemDetails := tools.GetLimitedData()
	var itemsWithin []Item
	for id, _ := range itemDetails.Items {
		name := itemDetails.Items[id][0]
		rap := itemDetails.Items[id][2].(float64)
		demand := int(itemDetails.Items[id][5].(float64))
		price := rap

		//Filter out items outside price range and demand
		if priceLow <= price && price <= priceHigh && (!isDemand || demand != -1) {
			priceFuture := projectPrice_FourierSTL(id, daysPast, daysFuture, config.LogConsole)
			z_score := findZScore(id, priceFuture, config.LogConsole)
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

// Scans z-scores of items within price range and demand level
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

// Scans items under z-score threshold within price range and demand level in lookback period
func SearchFallingItems(z_high float64, priceLow float64, priceHigh float64, isDemand bool) []string {
	return SearchItemsWithin(-9999, z_high, priceLow, priceHigh, isDemand)
}

// Analyzes the z-scores of inventory items and prints list of metrics
func AnalyzeInventory(forecastPrices bool, forecastType string) {
	assetIds := tools.GetInventory(fmt.Sprintf("%d", config.RobloxId))
	itemDetails := tools.GetLimitedData()
	var tot_z float64      //Total z-score
	var weighted_z float64 //Weighted z-score
	var tot_rap float64    //Total RAP
	var itemsProcessed int //# of items successfully processed
	fmt.Println("____________________________________________________")
	for _, id := range assetIds {
		if len(itemDetails.Items[id]) == 0 {
			continue
		}
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
	if forecastPrices {
		var tot_past_z float64      //Total forecasted z-scores
		var weighted_past_z float64 //Weighted forecasted z-scores
		fmt.Println("Forecasts:")
		for _, id := range assetIds {
			if len(itemDetails.Items[id]) == 0 {
				continue
			}
			name := itemDetails.Items[id][0]
			rap := itemDetails.Items[id][2].(float64)

			var past_z_score float64
			var priceFuture float64

			if forecastType == "z_score" {
				//Examine z-score from 2 months last year compared to its preceding 30 days
				past_z_score, priceFuture = projectPrice_ZScore(id, 330, 270, 450, 360, config.LogConsole)
				if past_z_score != past_z_score {
					continue //Check for NaN
				}
			} else if forecastType == "stl" {
				//Forecast future prices with STL + Fourier regression
				priceFuture = float64(projectPrice_FourierSTL(id, 365 * 3, 60, false))
				past_z_score = float64(findZScore(id, float64(priceFuture), false))
			}

			tot_past_z += past_z_score
			weighted_past_z += rap * past_z_score
			fmt.Println(name, "| Forecast Z-Score:", past_z_score, "| Price Prediction:", priceFuture)
		}
		fmt.Println()
		fmt.Println("Avg. Forecast Z-Score: ", (tot_past_z / float64(itemsProcessed)), " | ", "Weighted Forecast Z-Score: ", (weighted_past_z / float64(tot_rap)))
		fmt.Println("____________________________________________________")
	}
	fmt.Println("Listed Items: ", fmt.Sprintf("%d", itemsProcessed)+"/"+fmt.Sprintf("%d", len(assetIds)))
}

//Estimates item exchange value by projecting item prices with STL-Fourier
func EvaluateTrade(giveIds []string, receiveIds []string, daysPast int, daysFuture int) {
	itemDetails := tools.GetLimitedData()

	var forecast = func(id string) float64 {
		name := itemDetails.Items[id][0]

		//Forecast prices with STL decomposition
		priceSTL := projectPrice_FourierSTL(id, 365 * 3, 30, true)
		z_score_stl := findZScore(id, priceSTL, false)
		log.Println(name, "(STL) | Z-Score:", z_score_stl, "| Price Prediction:", priceSTL)
		
		return priceSTL
	}

	var giveValue, receiveValue float64
	for _, id := range giveIds {
		giveValue += forecast(id)
	}
	log.Println("____________________________________________________")
	for _, id := range receiveIds {
		receiveValue += forecast(id)
	}
	log.Println("____________________________________________________")
	log.Printf("Predicted Trade Value (%v Days)", daysFuture)
	log.Println("You Give:", giveValue)
	log.Println("You Receive:", receiveValue)
	log.Println()
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
					historyData = tools.SalesData[id]
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

				if historyData != nil {
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
