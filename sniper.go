package main

import (
	"context"
	"log"
	"robolimited/config"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Helper method for logging into account
func robloxLogin(ctx context.Context) error {
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

// Orders purchase on an item given id, checks best price against presumed RAP / value and returns success
func OrderPurchase(id string) bool {
	url := config.RobloxCatalogBaseURL + id
	priceSelector := config.PriceSelector
	buySelector := config.BuyButtonSelector
	confirmSelector := config.ConfirmButtonSelector
	timeoutSec := 15

	// chrome flags: headful & use profile to reuse login
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// Use headful so you can see what's happening
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-sandbox", true),

		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions-except", ""),
		chromedp.Flag("disable-plugins-discovery", ""),
		chromedp.Flag("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	// Create Chrome contexts
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Log into account
	log.Println("Logging into Roblox account...")
	err := robloxLogin(ctx)
	if err != nil {
		log.Printf("Login failed: %v\n", err)
		return false
	}

	// Navigate to item page
	log.Println("Navigating to item page...")

	err = chromedp.Run(ctx,
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

	itemDetails := GetLimitedData()
	info := itemDetails.Items[id]
	name := info[0].(string)
	RAP := int(info[2].(float64))
	value := int(info[3].(float64))
	demand := int(info[5].(float64))
	projected := int(info[7].(float64))

	log.Println("Comparing", bestPrice_r, "to", "RAP", RAP, "and value", value)
	if !BuyCheck(bestPrice, RAP, value, demand != -1) || projected != -1 || (config.StrictBuyCondition && !CheckDip(id, float64(bestPrice))) { //Failed price validation!
		log.Println("Failed price validation! Canceling..")
		return false
	}

	//Click purchase
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
	time.Sleep(250 * time.Millisecond)

	err = chromedp.Run(ctx, //Confirm click
		chromedp.WaitVisible(confirmSelector, chromedp.ByQuery),
		chromedp.Sleep(250*time.Millisecond),
		chromedp.Click(confirmSelector, chromedp.NodeVisible, chromedp.ByQuery),
	)

	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	log.Println("Executed buy order on:", name, id)

	time.Sleep(5 * time.Second)

	return true
}
