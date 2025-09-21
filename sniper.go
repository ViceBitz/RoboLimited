package main

import (
	"context"
	"log"
	"math"
	"robolimited/config"
	"robolimited/tools"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Global Chrome browser for fast page navigation
var (
	globalBrowser *tools.Browser
	browserOnce   sync.Once
	browserErr    error
)

// Logs into Roblox account
func robloxLogin(ctx context.Context) error {
	log.Println("Logging into Roblox account...")
	return chromedp.Run(ctx,
		// First navigate to Roblox
		chromedp.Navigate("https://www.roblox.com"),

		// Set the authentication cookie
		chromedp.ActionFunc(func(ctx context.Context) error {
			return network.SetCookies([]*network.CookieParam{
				{
					Name:     ".ROBLOSECURITY",
					Value:    config.RobloxCookie,
					Domain:   ".roblox.com",
					Path:     "/",
					Secure:   true,
					HTTPOnly: true,
				},
			}).Do(ctx)
		}),

		// Reload to apply the cookie
		chromedp.Reload(),

		// Wait for logged-in state (user avatar or menu)
		chromedp.WaitVisible(".avatar", chromedp.ByQuery),
	)

}

// Executes purchase on an item given id, checks best price against presumed RAP / value and returns success
func ExecutePurchase(id string, expectedPrice int) bool {
	url := config.RobloxCatalogBaseURL + id
	priceSelector := config.PriceSelector
	buySelector := config.BuyButtonSelector
	confirmSelector := config.ConfirmButtonSelector

	ctx, cancel := globalBrowser.GetContextWithTimeout(15 * time.Second)
	defer cancel()

	// Navigate to item page
	log.Println("Navigating to item page...")

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	//Final validation on best price against RAP / Value and projected status
	log.Println("Validating best price...")
	var bestPrice_r string
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(priceSelector, chromedp.ByQuery),
		chromedp.Text(priceSelector, &bestPrice_r, chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	bestPrice_r = strings.ReplaceAll(bestPrice_r, ",", "")
	bestPrice, _ := strconv.Atoi(bestPrice_r)

	log.Println("Comparing listed price", bestPrice_r, "to expected price", expectedPrice)
	//Must be within 10 robux of price error
	if math.Abs(float64(expectedPrice-bestPrice)) > 10 {
		log.Println("Failed price validation! Canceling..")
		return false
	}

	//Click purchase
	time.Sleep(250 * time.Millisecond) //Give webpage some time to load
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(buySelector, chromedp.ByQuery),
		chromedp.Click(buySelector, chromedp.NodeVisible, chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	log.Println("Clicked buy button")

	// Wait for purchase modal to appear (a bit of extra time for network)
	time.Sleep(500 * time.Millisecond)

	err = chromedp.Run(ctx, //Confirm click
		chromedp.WaitVisible(confirmSelector, chromedp.ByQuery),
		chromedp.Click(confirmSelector, chromedp.NodeVisible, chromedp.ByQuery),
	)

	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	log.Println("Executed buy order on:", id)

	time.Sleep(5 * time.Second)

	return true
}

// Initialize global browser instance on package initialization
func init() {
	browserOnce.Do(func() {
		globalBrowser, browserErr = tools.NewBrowser()
		//Log into Roblox account on startup
		err := robloxLogin(globalBrowser.GetContext())
		if err != nil {
			log.Printf("Login failed: %v\n", err)
		}

		if err != nil || browserErr != nil {
			log.Printf("Failed to initialize global browser: %v", browserErr)
		} else {
			log.Println("Global browser initialized and ready for use")
		}
	})
}
