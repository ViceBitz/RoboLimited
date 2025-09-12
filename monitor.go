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
	"strings"
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

func buyCheck(bestPrice int, RAP_r int, value_r int) bool {
	if bestPrice == 0 { //Error occurred if price is 0
		return false
	}
	RAP := float64(RAP_r)
	value := float64(value_r)
	if value == -1 {
		//RAP limited
		return bestPrice <= int(RAP*(1.0-config.RAPDipF))
	} else {
		//Value limited
		return bestPrice <= int(RAP*(1.0-config.RAPDipF)) || bestPrice <= int(value*(1.0-config.ValueDipF))
	}
}

func sellCheck(boughtPrice_r int, bestPrice_r int, value_r int) bool {
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

func getPriceInBatch(itemIDs []string) map[string]string {
	pool := tools.NewScraperPool(3) // 5 Chrome instances
	defer pool.Close()

	concurrentResults := pool.ProcessConcurrent(itemIDs)
	return concurrentResults
}

func getLimitedData() *ItemDetails {
	//Rolimons API endpoint for item details
	apiURL := config.RolimonsAPI

	//Make a GET request to the Rolimons API
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Fatalf("Error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	//Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	//Unmarshal JSON response into the ItemDetails struct
	var itemDetails ItemDetails
	err = json.Unmarshal(body, &itemDetails)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return &itemDetails
}

// Monitors the prices of limiteds by scanning limited site pages directly
func monitorDirectly() {
	var tradeSim *tools.TradeSimulator = tools.NewTradeSimulator()

	//id -> [item_name, acronym, rap, value, default_value, demand, trend, projected, hyped, rare]
	itemDetails := getLimitedData()

	//Filter out projected and low demand items
	targetItems := []string{}
	for id, _ := range itemDetails.Items {
		info := itemDetails.Items[id]
		//name := info[0].(string)
		RAP := int(info[2].(float64))
		//value := int(info[3].(float64))
		demand := int(info[5].(float64))
		projected := int(info[7].(float64))

		//Check within price range & demand
		if RAP >= config.PriceRangeLow && RAP <= config.PriceRangeHigh && projected == -1 && (!config.HighDemand || demand != -1) {
			targetItems = append(targetItems, id)
		}
		if len(targetItems) >= config.MaxLimiteds {
			break
		}
	}
	fmt.Printf("Total item count: %d\n", len(targetItems))
	//Monitor all targetted items for price drops
	for i := 0; i < 100; i++ {
		if i%config.ValueCycles == 0 {
			//Recalculate RAP / Value and limited data from Rolimon API
			itemDetails = getLimitedData()
		}
		fmt.Println("________________________________")

		//Get all best prices via multithreading (in batches)
		results := getPriceInBatch(targetItems)

		best_prices := make(map[string]int) //Convert sync map to concrete
		for _, id := range targetItems {
			results[id] = strings.ReplaceAll(results[id], ",", "")
			numPrice, _ := strconv.Atoi(results[id])
			best_prices[id] = numPrice
		}

		//Make decisions to buy/sell
		for _, id := range targetItems {
			info := itemDetails.Items[id]
			name := info[0].(string)
			RAP := int(info[2].(float64))
			value := int(info[3].(float64))

			best_price := best_prices[id]
			//Check buys
			if buyCheck(best_price, RAP, value) {
				//BUY
				tradeSim.BuyItem(id, name, best_price)
			}
			//Check sells
			for _, bought_price := range tradeSim.GetPortfolio()[id] {
				if sellCheck(bought_price, best_price, value) {
					//SELL
					tradeSim.SellItem(id, name, best_price, value)
					continue
				}
			}
		}

		time.Sleep(180 * time.Second)
	}
}

func getDealsData() *DealDetails {
	//Rolimons API for deal data
	dealURL := config.RolimonsDeals

	//Make a GET request to API
	resp, err := http.Get(dealURL)
	if err != nil {
		log.Fatalf("Error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	//Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	//Unmarshal JSON response into DealDetails struct
	var dealDetails DealDetails
	err = json.Unmarshal(body, &dealDetails)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return &dealDetails
}

// Monitor limited deals via Rolimon's deals page
func monitorDeals() {
	var tradeSim *tools.TradeSimulator = tools.NewTradeSimulator()

	//id -> [item_name, acronym, rap, value, default_value, demand, trend, projected, hyped, rare]
	itemDetails := getLimitedData()

	RAP_map := map[string]int{}

	for {
		//[[timestamp, isRAP, id, bestPrice / RAP]]
		dealDetails := getDealsData()
		activities := dealDetails.Activities

		for _, info := range activities {
			isRAP := int(info[1].(float64))
			id_r := int(info[2].(float64))
			id := strconv.Itoa(id_r)
			price := int(info[3].(float64))

			if len(itemDetails.Items[id]) == 0 {
				continue
			} //Handle not found error

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
				if buyCheck(price, RAP_map[id], value) {
					//BUY
					tradeSim.BuyItem(id, name, price)
				}
				//Check sells
				for _, bought_price := range tradeSim.GetPortfolio()[id] {
					if sellCheck(bought_price, price, value) {
						//SELL
						tradeSim.SellItem(id, name, price, value)
						continue
					}
				}

			} else { //Updating RAP
				RAP_map[id] = price
			}

			fmt.Println("Scanned", name, "|", RAP_map[id], value, price)
		}

		time.Sleep(time.Second * 3)
	}
}

// Driver
func main() {
	//monitorDirectly()
	monitorDeals()
}
