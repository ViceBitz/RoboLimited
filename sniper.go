package main

import (
	"log"
	"robolimited/config"
	"robolimited/tools"
	"strings"
	"net/http"
	"fmt"
	"encoding/json"
	"errors"
	"bytes"
	"io"
    "os"
    "github.com/google/uuid"
)


var CSRFToken string = ""
var consoleLog *os.File

type PurchasePayload struct {
    CollectibleItemId         string  `json:"collectibleItemId"`
    CollectibleItemInstanceId string  `json:"collectibleItemInstanceId"`
    CollectibleProductId      string  `json:"collectibleProductId"`
    ExpectedCurrency          int     `json:"expectedCurrency"`
    ExpectedPrice             int64   `json:"expectedPrice"`
    ExpectedPurchaserId       int64   `json:"expectedPurchaserId"`
    ExpectedPurchaserType     string  `json:"expectedPurchaserType"`
    ExpectedSellerId          int64   `json:"expectedSellerId"`
    ExpectedSellerType        string `json:"expectedSellerType"`
    IdempotencyKey            string  `json:"idempotencyKey"`
}

//Retrieves CSRF token for later use
func getCSRFToken(collectibleItemId string, cookie string, payload PurchasePayload) error {
    url := fmt.Sprintf(config.PurchaseAPI, collectibleItemId)
    client := tools.GlobalClient

    bodyData, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyData))
    if err != nil {
        return err
    }

    tools.FastHeaders(req)
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s", cookie))
    

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    //Handle CSRF token protection
    if resp.StatusCode == 403 {
        CSRFToken = resp.Header.Get("x-csrf-token")
        if CSRFToken == "" {
            return errors.New("no CSRF token found in 403 response")
        }
    }
    
    return nil
}

//Purchases item by making request to API endpoint
func purchaseItem(collectibleItemId string, cookie string, payload PurchasePayload, retry bool) error {
    //Write status to log file
    log.SetOutput(consoleLog)
    defer log.SetOutput(os.Stderr)

    url := fmt.Sprintf(config.PurchaseAPI, collectibleItemId)
    client := tools.GlobalClient

    bodyData, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    //Make POST request with current token
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyData))
    if err != nil {
        return err
    }

    tools.FastHeaders(req)
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s", cookie))
    req.Header.Set("X-CSRF-TOKEN", CSRFToken)

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    //Generate new X-CSRF token if invalid
    if resp.StatusCode == 403 {
        getCSRFToken(collectibleItemId, cookie, payload)
        if !retry {
            log.Println("Could not get X-CSRF token.")
            return err
        }
        return purchaseItem(collectibleItemId, cookie, payload, false)
    }

    if resp.StatusCode != 200 && resp.StatusCode != 201 {
		log.Printf("Purchase failed: status %d, response %s \n", resp.StatusCode, string(respBody))
        return err
    }

    if strings.Contains(string(respBody), "errors") {
		log.Printf("Purchase API error: %s \n", string(respBody))
        return err
    }

    log.Println("Purchase request executed:", string(respBody))
    return nil
}

//Executes purchase on an item via API call to economy endpoint
func ExecutePurchase(id string, bypass bool, value float64, isDemand bool) bool {
    cookie := config.RobloxCookie
    collectibleItemId, _ := tools.GetCollectibleId(id)
	sellers, err := tools.GetResellers(collectibleItemId)

    //Write status to log file
    log.SetOutput(consoleLog)
    defer log.SetOutput(os.Stderr)
    
	if err != nil {
		log.Println("Could not get reseller data:", err)
		return false
	}
	topSeller := sellers[0]

	//Validate actual price with expected
	if bypass || CheckDip(id, float64(topSeller.Price), value, isDemand) {
		//Request purchase using HTTP POST with payload
		payload := PurchasePayload{
            CollectibleItemId: collectibleItemId,
            CollectibleItemInstanceId: topSeller.CollectibleItemInstanceID,
            CollectibleProductId: topSeller.CollectibleProductID,
			ExpectedCurrency: 1,
			ExpectedPrice:    int64(topSeller.Price),
            ExpectedPurchaserId: config.RobloxId,
            ExpectedPurchaserType: "User",
            ExpectedSellerId: topSeller.Seller.SellerId,
			ExpectedSellerType: "User",
            IdempotencyKey: uuid.New().String(),
		}
		err := purchaseItem(collectibleItemId, cookie, payload, true)
		if err != nil {
			log.Println("Error making purchase:", err)
			return false
		}
		return true
	} else {
        log.Println("Price of", topSeller.Price, "does not match.")
    }

	return false
}

//Initialize tokens
func init() {
    //Set log to file
    consoleLog, _ = os.OpenFile(config.ConsoleLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    
    //Make dummy purchase for X-CSRF token
    ExecutePurchase("21070012", true, -1, false)
}