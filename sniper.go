package main

import (
	"log"
	"math"
	"robolimited/config"
	"robolimited/tools"
	"strings"
	"net/http"
	"fmt"
	"encoding/json"
	"errors"
	"bytes"
	"io"
)




type PurchasePayload struct {
    ExpectedCurrency int   `json:"expectedCurrency"`
    ExpectedPrice    int64 `json:"expectedPrice"`
    ExpectedSellerId int64 `json:"expectedSellerId"`
}

func purchaseItem(assetId string, cookie string, payload PurchasePayload) error {
    url := fmt.Sprintf("https://economy.roblox.com/v1/purchases/products/%s", assetId)

    client := &http.Client{}

    bodyData, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyData))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s", cookie))
    req.Header.Set("User-Agent", config.UserAgent)

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Handle CSRF token protection
    if resp.StatusCode == 403 {
        // Get CSRF Token from headers
        csrfToken := resp.Header.Get("x-csrf-token")
        if csrfToken == "" {
            return errors.New("no CSRF token found in 403 response")
        }

        // Retry with token
        req, err = http.NewRequest("POST", url, bytes.NewBuffer(bodyData))
        if err != nil {
            return err
        }
        req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s", cookie))
		req.Header.Set("User-Agent", config.UserAgent)
        req.Header.Set("X-CSRF-TOKEN", csrfToken)

        resp, err = client.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()
    }

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
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

// Executes purchase on an item via API call to economy endpoint
func ExecutePurchase(id string, expectedPrice int) bool {
    cookie := config.RobloxCookie
	sellers, err := tools.GetResellers(id)
	if err != nil {
		log.Println("Could not get reseller data:", err)
		return false
	}
	topSeller := sellers[0]

	//Validate actual price with expected
	if math.Abs(float64(topSeller.Price - expectedPrice)) < 10 {
		//Request purchase using HTTP POST with payload
		payload := PurchasePayload{
			ExpectedCurrency: 1,
			ExpectedPrice:    int64(topSeller.Price),
			ExpectedSellerId: topSeller.Person.SellerId,
		}
		err := purchaseItem(id, cookie, payload)
		if err != nil {
			log.Println("Error making purchase:", err)
			return false
		}
	}

	return true
}