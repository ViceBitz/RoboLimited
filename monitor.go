package main

import (
	"log"
	"math"
	"robolimited/config"
	"robolimited/tools"
	"strconv"
	"time"
)

/*
Monitors economy for price anomalies and market fluctuations through API requests.
Integrates the analyzer and sniper to detect dips and execute purchases.
*/

// Evaluates if margins are good enough to buy
func BuyF(rap_margin float64, value_margin float64, hasValue bool, isDemand bool) bool {
	//Implement demand evaluation (higher demand items have lower margin standards)
	if isDemand {
		if hasValue {
			return value_margin >= config.MarginD
		}
		return rap_margin >= config.MarginD
	} else {
		if hasValue {
			return value_margin >= config.MarginND
		}
		return rap_margin >= config.MarginND
	}
}

// Make decision on whether to buy or stand
func BuyCheck(bestPrice int, RAP_r int, value_r int, isDemand bool) bool {
	if bestPrice == 0 { //Error occurred or no resellers if price is 0
		return false
	}
	RAP := float64(RAP_r)
	value := float64(value_r)
	bpF := float64(bestPrice)
	if value == -1 {
		//RAP limited
		return BuyF((RAP-bpF)/RAP, -1, false, isDemand)
	} else {
		//Value limited
		return BuyF((RAP-bpF)/RAP, (value-bpF)/value, true, isDemand)
	}
}

// Throttles deal polling to avoid rate limit
func throttleMonitor() {
	//Sync throttle to unix offset for staggered scheduling
	var interval int64 = config.MonitorThrottle
	offset := time.Now().UnixMilli() % interval
	yieldTime := ((config.ClockOffset - offset) + interval) % interval

	//Ensure yield time does not drop below min
	if yieldTime < config.MinThrottle {
		yieldTime += interval
	}
	time.Sleep(time.Duration(yieldTime) * time.Millisecond)
}

// Monitor limited deals via Rolimon's deals page
func monitorDeals(live_money bool) {
	 //Make dummy purchase for X-CSRF token
	 ExecutePurchase("21070012", true, -1, false)

	var tradeSim *tools.TradeSimulator = tools.NewTradeSimulator()

	//id -> [item_name, acronym, rap, value, default_value, demand, trend, projected, hyped, rare]
	itemDetails := tools.GetLimitedData()

	RAP_map := map[string]int{}

	for i := range config.TotalIterations {

		//Bind throttle to unix timemark
		throttleMonitor()

		if config.LogConsole {
			log.Println("____________________________________________________")
		}

		if i%config.RefreshRate == 0 {
			//Recalculate RAP / Value and limited data from Rolimon API
			itemDetailsNew := tools.GetLimitedData()
			if itemDetailsNew == nil {
				//Mark errors in updating
				log.Println("Could not refresh item details..")
			} else {
				itemDetails = itemDetailsNew
			}
		}

		//[[timestamp, isRAP, id, bestPrice / RAP]]
		dealDetails := tools.GetDealsData()
		if dealDetails == nil {
			continue
		} //Catch error, wait for resolution

		activities := dealDetails.Activities

		for _, info := range activities {
			isRAP := int(info[1].(float64))
			id_r := int(info[2].(float64))
			id := strconv.Itoa(id_r)
			price := int(info[3].(float64))

			//Handle not found error
			if itemDetails == nil || len(itemDetails.Items[id]) == 0 {
				continue
			}

			isDemand := int(itemDetails.Items[id][5].(float64)) != -1
			projected := int(itemDetails.Items[id][7].(float64))

			//Exclude projected items and erroneous listings
			if projected != -1 || price < 1 {
				continue
			}
			//Exclude items out of price range
			if !(config.PriceRangeLow <= price && price <= config.PriceRangeHigh) {
				continue
			}

			//Scan for item details
			name := itemDetails.Items[id][0].(string)
			value := int(itemDetails.Items[id][3].(float64))
			_, inMap := RAP_map[id]
			if !inMap {
				RAP_map[id] = int(itemDetails.Items[id][2].(float64))
			}

			//Exclude items out of RAP range
			if !(config.RAPRangeLow <= RAP_map[id] && RAP_map[id] <= config.RAPRangeHigh) {
				continue
			}

			if isRAP == 0 { //Updating best price
				//Make decision to purchase item

				//Initial % margin filter of current price and RAP
				if !BuyCheck(price, RAP_map[id], value, isDemand) {
					continue
				}

				//Deeper price anomaly dip check using z-score below % margins
				if CheckDip(id, float64(price), float64(value), isDemand) {
					//BUY
					if live_money {
						ExecutePurchase(id, false, float64(value), isDemand)
					}
					tradeSim.BuyItem(id, name, price)
				}

				if config.LogConsole {
					log.Println("Scanned", name, "|", "RAP:", RAP_map[id], "| Value:", value, "| Price: ", price, "| Deal: ", math.Round(float64(max(RAP_map[id], value)-price)/float64(max(RAP_map[id], value))*1000.0)/10.0, "%")
				}

			} else { //Updating RAP
				RAP_map[id] = price

				if config.LogConsole {
					log.Println("Updated", name, "|", "RAP:", RAP_map[id], "| Value:", value, "| Price: ", price)
				}
			}
		}
	}
}

// Driver
func main() {
	//Start deal sniper process
	//monitorDeals(config.LiveMoney)

	//===Analyzer Methods===\\

	//Displays player inventory metrics
	AnalyzeInventory(true, "stl")

	//Check singular item's price trend with z-score
	//log.Println(findZScore("2620478831", 350, false)) //Check an item's current trend
	
	//Finds current price-lowering items in market
	//SearchFallingItems(-0.5, 2000, 6000, true) 

	//Forecast growth potential with z-score analysis of past year
	//SearchDatedWithin(-1000, 1000, 2000, 6000, 330, 270, 450, 360, true)
	
	/*
	itemDetails := tools.GetLimitedData()
	onlyDemand := false //scan demand items only
	forecastItems := []string{"9255011"} //"928908332", "20573078"
	for _,id := range(forecastItems) {
		name := itemDetails.Items[id][0]
		isDemand := int(itemDetails.Items[id][5].(float64)) != -1
		if !onlyDemand || isDemand {
			log.Println("____________________________________________________")
			//Forecast prices with z-score analysis
			z_score, priceFuture := projectPrice_ZScore(id, 330, 270, 450, 360, false)
			log.Println(name, "(Z-Score) | Z-Score:", z_score, "| Price Prediction:", priceFuture)

			//Forecast prices with STL decomposition
			priceSTL := projectPrice_STL(id, 400, 60, true)
			z_score_stl := findZScore(id, priceSTL, false)
			log.Println(name, "(STL) | Z-Score:", z_score_stl, "| Price Prediction:", priceSTL)
		
		}
	}
	*/
	
}
