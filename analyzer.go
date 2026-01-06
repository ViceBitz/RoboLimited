package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand/v2"
	"regexp"
	"robolimited/config"
	"robolimited/tools"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

/**
Toolkit/API for ingesting, transforming, and analyzing historical price data.
Uses statistical methods like z-score and STL-Fourier regression for forecasting.
Drives decision-making and guides buying/selling/trading. Used across different
components in this project, both automatically and manually.
*/

// Extracts time-series sales data from Rolimon's asset URL
func extractPriceSeries(url string) (*tools.Sales, error) {
	//Extract raw HTML from item page source
	html, _ := tools.GetPageSource(url)

	//Find sales data embedded within source using regex search
	re := regexp.MustCompile(`var\s+sales_data\s*=\s*(\{[\s\S]*?\});`)
	match := re.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("sales data not found in page HTML")
	}
	salesDataJSON := strings.TrimSuffix(match[1], ";")

	//Parse the actual sales data
	var salesData tools.Sales
	err := json.Unmarshal([]byte(salesDataJSON), &salesData)
	if err != nil {
		preview := salesDataJSON
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("failed to parse sales data JSON: %v\nJSON preview: %s", err, preview)
	}

	return &salesData, nil
}

// Extracts price data, resamples to 1-day snapshots, calculates mean/SD within date range
func processPriceSeries(id string, daysLower int64, daysUpper int64) (float64, float64, *tools.Sales, []int) {
	url := fmt.Sprintf(config.RolimonsSite, id)

	//Pull price data from cache if possible
	historyData := tools.SalesData[id]
	var err error
	if tools.SalesData[id] == nil {
		historyData, err = extractPriceSeries(url)
	}

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

// Extracts all owner ids of specific item from Rolimon's asset URL
func extractOwners(url string) ([]string, error) {
	//Extract raw HTML from item page source
	html, _ := tools.GetPageSource(url)

	//Find ownership data embedded within source using regex search
	re := regexp.MustCompile(`var\s+bc_copies_data\s*=\s*(\{[\s\S]*?\});`)
	match := re.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("bc_copies_data not found in page HTML")
	}

	//Parse the JSON
	jsonStr := strings.TrimSuffix(match[1], ";")
	var bcData struct {
		OwnerIDs []int64 `json:"owner_ids"`
		LastOnline []int64 `json:"bc_last_online"`
	}
	err := json.Unmarshal([]byte(jsonStr), &bcData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bc_copies_data JSON: %v", err)
	}

	//Sort by most recent active users online
	type bcUser struct {
		id string
		lastOnline int64
	}
	ownersData := make([]bcUser, len(bcData.OwnerIDs))
	for i := 0; i < len(bcData.OwnerIDs); i++ {
		ownersData[i] = bcUser{id: strconv.FormatInt(bcData.OwnerIDs[i], 10), lastOnline: bcData.LastOnline[i]}
	}
	sort.Slice(ownersData, func(i, j int) bool {
		return ownersData[i].lastOnline > ownersData[j].lastOnline
	})

	//Convert to string ids
	owners := make([]string, len(bcData.OwnerIDs))
	for i, u := range ownersData {
		owners[i] = u.id
	}

	return owners, nil
}

//Calculates z-score of price relative to designated origin
func findZScoreRelativeTo(id string, price float64, origin float64, logStats bool) float64 {
	_, std := tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev //Use cache for fast query
	if std == 0.0 { //Scrape mean and SD if not cached
		_, std, _, _ = processPriceSeries(id, config.LookbackPeriod, 0) //Get data from lookback period
	}
	z_score := (price - origin) / std

	if logStats {
		fmt.Println("Z-Score: ", z_score, "| Origin: ", origin, "| SD: ", std)
	}

	return z_score
}

//Calculates z-score of price relative to past sales data; pulls from cached data if exists
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
func modelZScore(id string, daysL1 int64, daysU1 int64, daysL2 int64, daysU2 int64, logStats bool) (float64, float64) {
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

Returns the forecasted price, residual standard dev. (to examine stability), peaks and dips timestamps
*/

func modelFourierSTL(id string, daysBefore int64, daysFuture int64, logStats bool) (float64, float64, []int, []int, []float64, []float64) {
	mean, _, _, pricePoints := processPriceSeries(id, daysBefore, 0)

	//Prepare price series for decomposition
	priceSeries := make([]float64, len(pricePoints))
	for i, v := range pricePoints {
		priceSeries[i] = float64(v)
	}
	n := len(priceSeries)

	if n < 20 {
		return math.NaN(), -1, make([]int, 0), make([]int, 0), make([]float64, 0), make([]float64, 0)
	}

	
	/*
	[[Fourier regression + linear drift to model seasonality]]

	(a) Fourier regression periodic function
	(b) Linear trendline
	(c) Raw price point scatterplot
	(d) Vertical dateline
	*/
	
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

	//Forecast future price average across set period
	ground := int((time.Now().Unix() - config.SalesDataOrigin) / (24 * 60 * 60)) //Days since data snapshot age
	horizon := int(daysFuture + int64(ground)) //Adjust for sales data age (predict avg. in [ground, daysFuture + ground])

	if daysFuture <= 0 {
		return math.NaN(), -1, make([]int, 0), make([]int, 0), make([]float64, 0), make([]float64, 0)
	}
	
	
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
		if (h >= 1+ground) { //Only include values after current date
			sumF += pred
		}
	}

	priceFuture := sumF / float64(daysFuture)

	//Recover coefficients from beta array
	idx := 0
	b0 := beta[idx]
	idx++
	bTrend := beta[idx]
	idx++

	bWeekly := beta[idx : idx+2*Kw]
	idx += 2 * Kw
	var bYearly []float64
	if useYearly {
		bYearly = beta[idx : idx+2*Ky]
	}
	//Dot product helper
	dot := func(a, b []float64) float64 {
		s := 0.0
		for i := 0; i < len(a); i++ {
			s += a[i] * b[i]
		}
		return s
	}

	/*
	[[Visualize Fourier regression model]]
	*/
	fitted := make(plotter.XYs, n)
	weeklySeason := make(plotter.XYs, n)
	var yearlySeason plotter.XYs
	if useYearly {
		yearlySeason = make(plotter.XYs, n)
	}
	rawPts := make(plotter.XYs, n)

	//Construct linear trendline
	trendLine := make(plotter.XYs, n)
	for t := 0; t < n; t++ {
		x := float64(t)
		trendVal := b0 + bTrend*(float64(t)/float64(n))

		trendLine[t].X = x
		trendLine[t].Y = trendVal
	}

	//Make plotting data
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

	//Graph Visualization

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

	//Add vertical dateline at offset from data age origin
	l, _ := plotter.NewLine(plotter.XYs{
		{X: float64(ground), Y: plt.Y.Min},
		{X: float64(ground), Y: plt.Y.Max},
	})
	plt.Add(l)
	plt.Legend.Add("Today", l)
	l.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	l.Width = vg.Points(1)

	//Set x-axis ticks to month (30 day intervals)
	var x_ticks []plot.Tick
	for i := 0; i <= int(daysBefore); i += 30 {
		x_ticks = append(x_ticks, plot.Tick{Value: float64(i), Label: fmt.Sprintf("%d", i)})
	}
	plt.X.Tick.Marker = plot.ConstantTicks(x_ticks)

	if err := plt.Save(vg.Length(daysBefore/365*300), 450, "data/fourier_stl_model.png"); err != nil {
		fmt.Println("plot save png:", err)
	}

	//Classify stable trends with residual component SD
	residuals := make([]float64, n)
	for t := 0; t < n; t++ {
		trendVal := b0 + bTrend*(float64(t)/float64(n))
		weeklyVal := dot(tools.FourierFeatures(t, 7.0, Kw), bWeekly)
		yearlyVal := 0.0
		if useYearly {
			yearlyVal = dot(tools.FourierFeatures(t, 365.25, Ky), bYearly)
		}
		fittedVal := trendVal + weeklyVal + yearlyVal
		residuals[t] = priceSeries[t] - fittedVal
	}
	var sumRes float64
	for _, r := range residuals {
		sumRes += r * r
	}
	residualSD := math.Sqrt(sumRes / float64(n)) //lower = stable

	//Calculate approx. amplitude for peak/dip check
	low := math.MaxFloat64
	high := -math.MaxFloat64

	for t := 1; t < min(n-1, 365); t++ {
		low = min(low, fitted[t].Y)
		high = max(high, fitted[t].Y)
	}
	amp := high - low

	/*
	[[Pinpoint peaks and dips in model in one cycle (rest are periodic)]]

	(1) Use derivative (dy/dx for adjacent points) to determine uphill vs. downhill
	(2) Check if magnitude of change is significant enough relative to amplitude 
	(3) Keep minimum spacing between consecutive peak/dip records
	(4) Ascend to peaks and descend to dips, store times
	*/

	peaks := []int{} //Peak times
	dips := []int{} //Dip times
	peak_ratios := []float64{} //Peak scale relative to mean
	dip_ratios := []float64{} //Dip scale relative to mean

	last_peak := -999 //Previous peak
	last_dip := -999 //Previous dip
	
	ascending := false //Ascending to peak
	descending := false //Descending to dip

	epsilon := 30 //neighbor band
	spacing := epsilon //minimum gap
	amp_min := 0.025 * float64(epsilon)/30 //% of amplitude to consider extrema
	running_mean := 0.0 //Running avg. of emitted prices

	for t := 1; t <= min(n-1, epsilon); t++ { running_mean += fitted[t].Y}

	for t := 1+epsilon; t < min(n-1, 365 * 2)-epsilon; t++ {
		running_mean += fitted[t].Y
		prev := fitted[t-epsilon].Y
		curr := fitted[t].Y
		next := fitted[t+epsilon].Y

		//Derivative & amplitude test
		if (curr > prev && curr > next &&
			math.Abs(curr-prev) > amp_min * amp && math.Abs(curr-next) > amp_min * amp){
			//Extend current ascending peak if moving uphill
			if ascending {
				adjustedT := t
				if (t > 365) {
					adjustedT = t - 365
				}
				peaks[len(peaks)-1] = adjustedT
				peak_ratios[len(peak_ratios)-1] = fitted[t].Y / (running_mean / float64(t))
				last_peak = t
				continue
			}
			//Insert new peak if changing directions (downhill to uphill)
			if (t - last_peak >= spacing) { //Spacing check
				//Second cycle: push back time and insert point into array 
				if (t > 365) {
					adjustedT := t - 365
					i := 0
					for i < len(peaks) && peaks[i] < adjustedT {
						i++
					}
					
					//Double check spacing while inserting
					left := i <= 0 || adjustedT - peaks[i-1] >= spacing
					right := i >= len(peaks) || peaks[i] - adjustedT >= spacing
					if left && right {
						peaks = slices.Insert(peaks, i, adjustedT)
						peak_ratios = slices.Insert(peak_ratios, i, fitted[t].Y / (running_mean / float64(t)))
						last_peak = t
						ascending = true
					}
				//First cycle: directly add point
				} else {
					peaks = append(peaks, t)
					peak_ratios = append(peak_ratios, fitted[t].Y / (running_mean / float64(t)))
					last_peak = t
					ascending = true
				}
			}
			descending = false
		}
		//Derivative & amplitude test
		if (curr < prev && curr < next &&
			math.Abs(curr-prev) > amp_min * mean && math.Abs(curr-next) > amp_min * mean) {
			//Extend current descending dip if moving downhill
			if descending {
				adjustedT := t
				if (t > 365) {
					adjustedT = t - 365
				}
				dips[len(dips)-1] = adjustedT
				dip_ratios[len(dip_ratios)-1] = fitted[t].Y / (running_mean / float64(t))
				last_dip = t
				continue
			}
			//Insert new dip if changing directions (uphill to downhill)
			if (t - last_dip >= spacing) { //Spacing check
				//Second cycle: push back time and insert point into array 
				if (t > 365) {
					adjustedT := t - 365
					i := 0
					for i < len(dips) && dips[i] < adjustedT {
						i++
					}
					//Double check spacing while inserting
					left := i <= 0 || adjustedT - dips[i-1] >= spacing
					right := i >= len(dips) || dips[i] - adjustedT >= spacing
					if left && right {
						dips = slices.Insert(dips, i, adjustedT)
						dip_ratios = slices.Insert(dip_ratios, i, fitted[t].Y / (running_mean / float64(t)))
						last_dip = t
						descending = true
					}
					
				//First cycle: directly add point
				} else {
					dips = append(dips, t)
					dip_ratios = append(dip_ratios, fitted[t].Y / (running_mean / float64(t)))
					last_dip = t
					descending = true
				}
			}
			ascending = false
		}
		//Reset neighbor extremas if entering into repeating section (i.e. at 365)
		if (t == 365) {
			last_peak = -1
			last_dip = -1
			ascending = false
			descending = false
		}
	}

	/*
	[[Date Filtering:]]

	(1) Adjust peaks/dips for sales data age & filter out old points beyond certain age (>45 days)
	
	(2) Flip values greater than half year from x -> x-365 to place in previous cycle 
	so first peak/dip will be proximal to current time (if beyond age, will be cut anyways)

	*/
	var peaks_filt []int
	var dips_filt []int
	maxAge := 90 //Ignore adjusted extrema further back than this
	log.Println(peaks, dips)
	for i := 0; i < len(peaks); i++ {
		peaks[i] -= ground
		if (peaks[i] > 365/2) {
			peaks[i] -= 365
		}
		if (peaks[i] >= -maxAge) { 
			peaks_filt = append(peaks_filt, peaks[i])
		}
	}
	for i := 0; i < len(dips); i++ {
		dips[i] -= ground
		if (dips[i] > 365/2) {
			dips[i] -= 365
		}
		
		if (dips[i] >= -maxAge) {
			dips_filt = append(dips_filt, dips[i])
		}
	}
	peaks = peaks_filt
	dips = dips_filt
	sort.Ints(peaks_filt)
	sort.Ints(dips_filt)

	return priceFuture, residualSD / mean, peaks, dips, peak_ratios, dip_ratios
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
type Prediction struct {
	id          string
	z_score     float64
	priceFuture float64
	nextPeak int
	nextDip int
	nextRatioP float64
	nextRatioD float64
	stability float64
}

/*
Searches for items of STL price forecast within price range, demand level, and date range
Also finds next peak and dip based on model and assesses overall item stability with
the %CV of the residue.

Orders items ascending by specified attribute, which includes:
- z-score: Z-score of forecasted price compared to past 90 days
- dips: Dip times
- peaks: Peak times
- d_ratio: Dip ratio to mean
- p_ratio: Peak ratio to mean
- stability: Stability (std. dev of residual values)
*/

func ForecastWithin(z_low float64, z_high float64, priceLow float64, priceHigh float64, daysPast int64, daysFuture int64, isDemand bool, sortBy string) []string {
	itemDetails := tools.GetLimitedData()
	var itemsWithin []Prediction
	for id, _ := range itemDetails.Items {
		name := itemDetails.Items[id][0]
		rap := itemDetails.Items[id][2].(float64)
		demand := int(itemDetails.Items[id][5].(float64))
		price := rap

		//Filter out items outside price range and demand
		if priceLow <= price && price <= priceHigh && (!isDemand || demand >= 1) {
			priceFuture, stability, peaks, dips, p_ratios, d_ratios := modelFourierSTL(id, daysPast, daysFuture, config.LogConsole)
			z_score := findZScoreRelativeTo(id, priceFuture, rap, config.LogConsole) //Get z-score relative to live RAP
			if z_low <= z_score && z_score <= z_high {
				nextPeak := -1; nextRatioP := 0.0
				if (len(peaks) > 0) { nextPeak = peaks[0]; nextRatioP = p_ratios[0]}
				nextDip := -1; nextRatioD := 0.0
				if (len(dips) > 0) { nextDip = dips[0]; nextRatioD = d_ratios[0]}
				itemsWithin = append(itemsWithin, Prediction{id, z_score, priceFuture, nextPeak, nextDip, nextRatioP, nextRatioD, stability})
			}
			fmt.Println("Processed item:", id, "| Z-Score:", z_score, "| Price Prediction:", priceFuture, "|", name)
		}

	}
	switch sortBy {
	//Sort by ascending z-score
	case "z-score":
		sort.Slice(itemsWithin, func(i, j int) bool {
			return itemsWithin[i].z_score < itemsWithin[j].z_score
		})
	//Sort by earliest dip
	case "dips":
		sort.Slice(itemsWithin, func(i, j int) bool {
			return float64(itemsWithin[i].nextDip) < float64(itemsWithin[j].nextDip)
		})
	//Sort by earliest peak
	case "peaks":
		sort.Slice(itemsWithin, func(i, j int) bool {
			return float64(itemsWithin[i].nextPeak) < float64(itemsWithin[j].nextPeak)
		})
	//Sort by dip ratios
	case "d_ratio":
		sort.Slice(itemsWithin, func(i, j int) bool {
			return float64(itemsWithin[i].nextRatioD) < float64(itemsWithin[j].nextRatioD)
		})
	//Sort by peak ratios
	case "p_ratio":
		sort.Slice(itemsWithin, func(i, j int) bool {
			return float64(itemsWithin[i].nextRatioP) < float64(itemsWithin[j].nextRatioP)
		})
	//Sort by stability
	case "stability":
		sort.Slice(itemsWithin, func(i, j int) bool {
			return float64(itemsWithin[i].stability) < float64(itemsWithin[j].stability)
		})
	}
	
	var onlyItems []string
	for _, m := range itemsWithin {
		name := itemDetails.Items[m.id][0]
		rap := itemDetails.Items[m.id][2]
		onlyItems = append(onlyItems, m.id)
		fmt.Println("Found item:", m.id, "| RAP:", rap, "| Z-Score:", math.Trunc(m.z_score*100)/100, "| Abs. Price Diff:", math.Trunc((m.priceFuture-rap.(float64))*100)/100, "|", name)
		fmt.Println("Peak:", m.nextPeak, "| Dip:", m.nextDip, "| Stability:", m.stability)
		fmt.Println("Peak Ratio:", m.nextRatioP, "| Dip Ratio:", m.nextRatioD)
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
		if priceLow <= price && price <= priceHigh && (!isDemand || demand >= 1) {
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
		fmt.Println("Found item:", m.id, "| Z-Score:", math.Trunc(m.z_score*100)/100, "|", name)
	}

	return onlyItems
}

// Scans items under z-score threshold within price range and demand level in lookback period
func SearchFallingItems(z_high float64, priceLow float64, priceHigh float64, isDemand bool) []string {
	return SearchItemsWithin(-9999, z_high, priceLow, priceHigh, isDemand)
}

// Looks for item owners within net worth range and construct trade links
func FindOwners(targetItemId string, worth_low float64, worth_high float64, limit int) {
	url := fmt.Sprintf(config.RolimonsSite, targetItemId)
	ownerIds, _ := extractOwners(url)

	itemDetails := tools.GetLimitedData()

	//Take recent slice, shuffle owners for a random picking
	ownerIds = ownerIds[:min(int(len(ownerIds)), limit * 20)]
	for i := range ownerIds {
		j := rand.IntN(i + 1)
		ownerIds[i], ownerIds[j] = ownerIds[j], ownerIds[i]
	}

	//Calculate net worth of every owner
	log.Println(len(ownerIds))
	for _, owner := range ownerIds {
		if limit <= 0 { //Don't go over link limit
			break
		}

		assetIds := tools.GetInventory(owner)
		netWorth := 0.0
		for _, id := range assetIds {
			if len(itemDetails.Items[id]) == 0 {
				continue
			}
			rap := itemDetails.Items[id][2].(float64)
			netWorth += rap
		}

		//Check if total RAP within net worth range
		if netWorth > 0 && worth_low <= netWorth && netWorth <= worth_high {
			log.Println("Trade Link:", fmt.Sprintf(config.PlayerTrade, owner), "| Net Worth:", netWorth)
			limit--
		}
		time.Sleep(2 * time.Second) //Avoid rate-limit on inventory scan
	}
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
		var tot_rap float64 //Total predicted item RAP values
		fmt.Println("Forecasts:")
		for _, id := range assetIds {
			if len(itemDetails.Items[id]) == 0 {
				continue
			}
			name := itemDetails.Items[id][0]
			rap := itemDetails.Items[id][2].(float64)

			var past_z_score float64

			if forecastType == "z_score" {
				//Examine z-score from 2 months last year compared to its preceding 30 days
				past_z_score, _ = modelZScore(id, 330, 270, 450, 360, config.LogConsole)
				if past_z_score != past_z_score {
					continue //Check for NaN
				}
			} else if forecastType == "stl" {
				//Forecast future prices with STL + Fourier regression
				priceSTL, stability, peaks, dips, p_ratios, d_ratios := modelFourierSTL(id, 365 * 4, 30, true)
				z_score_stl := findZScoreRelativeTo(id, priceSTL, rap, false)
				past_z_score = z_score_stl
				tot_rap += priceSTL
				
				peaks = append(peaks, -1); dips = append(dips, -1); p_ratios = append(p_ratios, -1); d_ratios = append(d_ratios, -1)

				fmt.Println(name, "|", id, "| Z-Score:", z_score_stl, "| RAP:", rap, "| Price Prediction:", priceSTL)
				fmt.Println("Peak:", peaks[0], "| Dip:", dips[0], "| Stability: ", stability)
				fmt.Println("Peak Ratio:", p_ratios[0], "| Dip Ratio:", d_ratios[0], "\n")
			}

			tot_past_z += past_z_score
			weighted_past_z += rap * past_z_score
			
		}
		fmt.Println("Predicted Portfolio Value: ", tot_rap)
		fmt.Println("Avg. Forecast Z-Score: ", (tot_past_z / float64(itemsProcessed)), " | ", "Weighted Forecast Z-Score: ", (weighted_past_z / float64(tot_rap)))
		fmt.Println("____________________________________________________")
	}
	fmt.Println("Listed Items: ", fmt.Sprintf("%d", itemsProcessed)+"/"+fmt.Sprintf("%d", len(assetIds)))
}

// Estimates item exchange value by projecting item prices with STL-Fourier
func EvaluateTrade(giveIds []string, receiveIds []string, daysPast int64, daysFuture int64) {
	itemDetails := tools.GetLimitedData()

	var forecast = func(id string) float64 {
		if len(itemDetails.Items[id]) == 0 {
			return 0
		}
		name := itemDetails.Items[id][0]

		//Forecast prices with STL decomposition
		priceFuture, _, _, _, _, _ := modelFourierSTL(id, daysPast, daysFuture, true)
		z_score_stl := findZScore(id, priceFuture, false)
		log.Println(name, "(STL) | Z-Score:", z_score_stl, "| Price Prediction:", priceFuture)

		return priceFuture
	}

	var giveValue, receiveValue float64
	log.Println("____________________________________________________")
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
	log.Println("____________________________________________________")
}

func init() {
	//Initialize global sales maps
	tools.SalesStats = tools.RetrieveSalesStats()
	tools.SalesData = tools.RetrieveSalesData()

	//Precompute mean & standard deviation for past sales data of all items
	//Write to a .csv file to use for querying later
	//Make sure to update SalesDataOrigin in settings.go
	if config.PopulateSalesData {
		cycleIncomplete := true //to check if data collection is complete

		for cycleIncomplete {
			//Refresh global sales cache
			tools.SalesStats = tools.RetrieveSalesStats()
			tools.SalesData = tools.RetrieveSalesData()

			log.Println("Sales Stats: ", len(tools.SalesStats))
			log.Println("Sales Data: ", len(tools.SalesData))

			cycleIncomplete = false
			itemDetails := tools.GetLimitedData()

			var sales_stats []tools.StatsPoint
			sales_data := make(map[string]*tools.Sales)

			var mu sync.Mutex
			var wg sync.WaitGroup

			//Multithread scan Rolimon's for sales data
			maxThreads := 4
			semaphore := make(chan struct{}, maxThreads)

			for id, _ := range itemDetails.Items {
				wg.Add(1)
				go func(itemID string) {
					defer wg.Done()

					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					//Try to pull from caches
					var historyData *tools.Sales

					mean, SD := 0.0, 0.0
					if tools.SalesStats[id].Mean != 0.0 { //stats cache
						mean, SD = tools.SalesStats[id].Mean, tools.SalesStats[id].StdDev
					} else {
						//Get data from lookback period
						mean, SD, historyData, _ = processPriceSeries(itemID, config.LookbackPeriod, 0)
						cycleIncomplete = true
					}

					if tools.SalesData[id] != nil { //data cache
						historyData = tools.SalesData[id]
					} else {
						if historyData == nil {
							_, _, historyData, _ = processPriceSeries(itemID, config.LookbackPeriod, 0)
						}
						cycleIncomplete = true
					}

					//Check throttle to prevent excessive rate-limiting
					if mean == 0.0 && SD == 0.0 {
						log.Println("Rate limited ... waiting")
						time.Sleep(15 * time.Second)
						cycleIncomplete = true
					}

					mu.Lock()

					sales_stats = append(sales_stats, tools.StatsPoint{ID: itemID, Mean: mean, StdDev: SD})
					log.Println("(", len(sales_stats), "/", len(itemDetails.Items), ")", "Reading sales stats of", itemID, "| Mean:", mean, "| SD:", SD)

					if historyData != nil {
						sales_data[itemID] = historyData
						log.Println("Reading sales data | Length: ", len(historyData.AvgDailySalesPrice))
					}

					mu.Unlock()
				}(id)
			}

			wg.Wait()
			log.Println("Caching stats and data into files..")
			tools.StoreSalesStats(sales_stats)
			tools.StoreSalesData(sales_data)
		}

	}
}
