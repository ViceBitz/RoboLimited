package main

import (
	"log"
	"math"
	"robolimited/config"
	"robolimited/tools"
	"strconv"
	"math/rand"
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
			return value_margin >= config.ValueDipD
		}
		return rap_margin >= config.RAPDipD
	} else {
		if hasValue {
			return value_margin >= config.ValueDipND
		}
		return rap_margin >= config.RAPDipND
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

// Monitor limited deals via Rolimon's deals page
func monitorDeals(live_money bool) {

	var tradeSim *tools.TradeSimulator = tools.NewTradeSimulator()

	//id -> [item_name, acronym, rap, value, default_value, demand, trend, projected, hyped, rare]
	itemDetails := tools.GetLimitedData()

	RAP_map := map[string]int{}

	for i := range config.TotalIterations {
		//Throttle with random jitters
		throttleDur := time.Duration(config.MonitorThrottle + rand.Intn(config.MonitorThrottle / 10)) * time.Millisecond
		time.Sleep(throttleDur)

		if (config.LogConsole) {
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

			demand := int(itemDetails.Items[id][5].(float64))
			projected := int(itemDetails.Items[id][7].(float64))

			//Exclude projected items
			if projected != -1 {
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

				//Quick % price filter to eliminate obvious non-anomalies
				if BuyCheck(price, RAP_map[id], value, demand != -1) {
					//Price anomaly dip check using z-score
					if CheckDip(id, float64(price), demand != -1) {
						//BUY
						if live_money {
							ExecutePurchase(id, false, demand != -1)
						}
						tradeSim.BuyItem(id, name, price)
					}
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

//Driver
func main() {
	//Start deal sniper process
	monitorDeals(config.LiveMoney)
	
	//Analyzer Methods
	//SearchFallingItems(-0.5, 5000, 13000, true) //Finds price-lowering items in market
	//log.Println(FindOptimalSell("21070090")) //Pinpoints optimal selling price
	//log.Println(findZScore("1428418448", 12748, false)) //Check an item's current trend

	//Order executor test
	//ExecutePurchase("331486631", true)
}
