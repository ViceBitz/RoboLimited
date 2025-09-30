package tools

import (
    "fmt"
    "io"
    "net/http"
	"robolimited/config"
	"encoding/json"
	"log"
)

/*
Handles all API requests to Roblox & Rolimons endpoints retrieving item details,
resellers, prices, and most recent deals
*/

var GlobalClient = &http.Client{}

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
	body, err := io.ReadAll(resp.Body)
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
	body, err := io.ReadAll(resp.Body)
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

type ResellerData struct {
	Data []ResellerResponse
}

type ResellerResponse struct {
	CollectibleProductID     string 
	CollectibleItemInstanceID string
	Seller                    Seller
	Price                     int
	SerialNumber              int64
	ErrorMessage              *string
}

type Seller struct {
	HasVerifiedBadge bool
	SellerId         int64
	SellerType       string
	Name             string
}

type collectibleResponse struct {
	CollectibleItemId string
	ProductId int64
}

//Retrieves collectible and product id of limited from its asset id
func GetCollectibleId(assetId string) (string, error) {
	url := fmt.Sprintf("https://catalog.roblox.com/v1/catalog/items/%s/details?itemType=Asset", assetId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s", config.RobloxCookie))
	req.Header.Set("User-Agent", config.UserAgent)
	
	client := GlobalClient
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.ReadAll(resp.Body)
		return "", err
	}
	var res collectibleResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if res.CollectibleItemId == "" {
		return "", err
	}
	return res.CollectibleItemId, nil
}

//Gets all resellers of an item
func GetResellers(assetId string) ([]ResellerResponse, error) {
	collectibleId, err := GetCollectibleId(assetId)
	if (err != nil) {
		log.Println(err);
	}
	url := fmt.Sprintf("https://apis.roblox.com/marketplace-sales/v1/item/%s/resellers?limit=1", collectibleId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s", config.RobloxCookie))
	req.Header.Set("User-Agent", config.UserAgent)

	client := GlobalClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get resellers: status %d, response %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	var data ResellerData
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Println(err)
	}
	return data.Data, nil
}