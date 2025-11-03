package tools

import (
    "fmt"
    "io"
    "net/http"
	"net/url"
	"robolimited/config"
	"encoding/json"
	"time"
	"math/rand"
	"log"
	"os"
	"bufio"
)

/*
Handles all API requests to Roblox & Rolimons endpoints retrieving item details,
resellers, prices, and most recent deals
*/

//HTTP client for non-urgent API requests
var GlobalClient = &http.Client{} 

//Proxies and headers for market monitoring
var proxies []*url.URL 
var proxyIndex int
var userAgents []string

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

	//Send request through cycled proxies
	client := GlobalClient
	if (config.RotateProxies) {
		proxyURL := proxies[proxyIndex]
		proxyIndex = (proxyIndex + 1) % len(proxies)

		transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		client = &http.Client{Transport: transport, Timeout: 15 * time.Second}
	}
	
	//Build GET request with random user agent
	req, err := http.NewRequest("GET", dealURL, nil)
	if err != nil {
		log.Println("Error building GET request: ", err)
	}
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	//Make GET request to API
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error making HTTP request:", err)
		return nil
	}
	defer resp.Body.Close()

	//Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return nil
	}

	//Unmarshal JSON response into DealDetails struct
	var dealDetails DealDetails
	err = json.Unmarshal(body, &dealDetails)
	if err != nil {
		log.Println("Error unmarshalling JSON:", err)
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
	url := fmt.Sprintf(config.AssetAPI, assetId)
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
func GetResellers(collectibleId string) ([]ResellerResponse, error) {
	url := fmt.Sprintf(config.ResellerAPI, collectibleId)
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

func init() {
	//Initialize user agents
	headerFile, err := os.Open(config.AgentsFile)
	if err != nil {
		log.Println("Unable to open agent file: ", err)
	}
	defer headerFile.Close()
	
	scanner := bufio.NewScanner(headerFile)
	for scanner.Scan() {
		userAgents = append(userAgents, scanner.Text())
	}

	//Initialize proxy URLs
	proxyFile, err := os.Open(config.ProxyFile)
	if err != nil {
		log.Println("Unable to open proxy file: ", err)
	}
	defer proxyFile.Close()

	scanner = bufio.NewScanner(proxyFile)

	var proxyUser string
	var proxyPass string
	for scanner.Scan() {
		//Set IP credentials for auth
		if proxyUser == "" {
			proxyUser = scanner.Text()
		} else if proxyPass == "" {
			proxyPass = scanner.Text()
		} else {
			//Generate authenticated proxy URLs
			proxyHost := "dc.oxylabs.io"
			port := scanner.Text()
			proxies = append(proxies, &url.URL{
				Scheme: "http",
				Host:   proxyHost + ":" + port,
				User:   url.UserPassword(proxyUser, proxyPass),
			})
		}
	}
	proxyIndex = 0
}