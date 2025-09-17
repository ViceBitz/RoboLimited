package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"robolimited/config"
	"robolimited/tools"
	"strconv"
	"time"
)

// ItemDetails JSON structure
type ItemDetails struct {
	ItemCount int                      `json:"item_count"`
	Items     map[string][]interface{} `json:"items"`
}

// DealDetails JSON structure
type DealDetails struct {
	Success    bool            `json:"success"`
	Activities [][]interface{} `json:"activities"`
}

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
		return BuyF((RAP-bpF)/bpF, -1, false, isDemand)
	} else {
		//Value limited
		return BuyF((RAP-bpF)/bpF, (value-bpF)/value, true, isDemand)
	}
}

/* DEPRECATED
func SellCheck(boughtPrice_r int, bestPrice_r int, value_r int) bool {
	value := float64(value_r)
	boughtPrice := float64(boughtPrice_r)
	bestPrice := float64(bestPrice_r)

	if value == -1 {
		//RAP limited
		return bestPrice*0.7-boughtPrice > boughtPrice*config.SellMargin
	} else {
		//Value limited
		return value*0.7-boughtPrice > boughtPrice*config.SellMargin
	}

}
*/

func GetLimitedData() *ItemDetails {
	//Rolimons API endpoint for item details
	apiURL := config.RolimonsAPI

	//Make a GET request to the Rolimons API
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error making HTTP request: %v", err)
		return nil
	}
	defer resp.Body.Close()

	//Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil
	}

	//Unmarshal JSON response into the ItemDetails struct
	var itemDetails ItemDetails
	err = json.Unmarshal(body, &itemDetails)
	if err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		return nil
	}

	return &itemDetails
}

func GetDealsData() *DealDetails {
	//Rolimons API for deal data
	dealURL := config.RolimonsDeals

	//Make a GET request to API
	resp, err := http.Get(dealURL)
	if err != nil {
		log.Printf("Error making HTTP request: %v", err)
		return nil
	}
	defer resp.Body.Close()

	//Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil
	}

	//Unmarshal JSON response into DealDetails struct
	var dealDetails DealDetails
	err = json.Unmarshal(body, &dealDetails)
	if err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		return nil
	}

	return &dealDetails
}

// Monitor limited deals via Rolimon's deals page
func monitorDeals(live_money bool) {
	var tradeSim *tools.TradeSimulator = tools.NewTradeSimulator()

	//id -> [item_name, acronym, rap, value, default_value, demand, trend, projected, hyped, rare]
	itemDetails := GetLimitedData()

	RAP_map := map[string]int{}

	for i := range 10000 {
		fmt.Println("____________________________________________________")
		if i%config.RefreshRate == 0 {
			//Recalculate RAP / Value and limited data from Rolimon API
			itemDetails = GetLimitedData()
			if itemDetails == nil {
				continue
			} //Catch error, wait for resolution
		}

		//[[timestamp, isRAP, id, bestPrice / RAP]]
		dealDetails := GetDealsData()
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
			if len(itemDetails.Items[id]) == 0 {
				continue
			}

			demand := int(itemDetails.Items[id][5].(float64))
			projected := int(itemDetails.Items[id][7].(float64))

			//Exclude projected items
			if projected != -1 {
				continue
			}
			//Exclude items out of price range
			if price >= config.PriceRangeLow && price <= config.PriceRangeHigh {
				continue
			}

			//Scan for item details
			name := itemDetails.Items[id][0].(string)
			value := int(itemDetails.Items[id][3].(float64))
			_, inMap := RAP_map[id]
			if !inMap {
				RAP_map[id] = int(itemDetails.Items[id][2].(float64))
			}

			if isRAP == 0 { //Updating best price
				//Make decision to buy/sell

				//Check buys
				if BuyCheck(price, RAP_map[id], value, demand != -1) {
					//Final price manipulation check
					if !CheckProjected(id, float64(RAP_map[id])) {
						//BUY
						if !live_money {
							tradeSim.BuyItem(id, name, price)
						} else {
							OrderPurchase(id)
						}
					}
				}
				/* DEPRECATED
				//Check sells
				for _, bought_price := range tradeSim.GetPortfolio()[id] {
					if SellCheck(bought_price, price, value) {
						//SELL
						tradeSim.SellItem(id, name, price, value)
						continue
					}
				}
				*/

				fmt.Println("Scanned", name, "|", "RAP:", RAP_map[id], "| Value:", value, "| Price: ", price)

			} else { //Updating RAP
				RAP_map[id] = price

				fmt.Println("Updated", name, "|", "RAP:", RAP_map[id], "| Value:", value, "| Price: ", price)
			}

		}

		time.Sleep(time.Second * 3)
	}
}

// Driver
func main() {
	//monitorDeals(config.LiveMoney)
	fmt.Println(CheckProjected("91294485", 42500.0))
}
