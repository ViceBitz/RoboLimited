package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"robolimited/config"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Copies file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

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
	url := flag.String("url", config.RobloxCatalogBaseURL+id, "Roblox catalog URL")
	priceSelector := flag.String("pricesel", config.PriceSelector, "CSS selector for best price")
	buySelector := flag.String("buysel", config.BuyButtonSelector, "CSS selector for Buy")
	confirmSelector := flag.String("confirmsel", config.ConfirmButtonSelector, "CSS selector for Confirm")
	timeoutSec := flag.Int("timeout", 15, "Timeout")

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

	ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutSec)*time.Second)
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

	time.Sleep(1000 * time.Millisecond)
	err = chromedp.Run(ctx,
		chromedp.Navigate(*url),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	//Final validation on best price against RAP / Value and projected status
	log.Println("Validating best price...")
	var bestPrice_r string
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(*priceSelector, chromedp.ByQuery),
		chromedp.Text(*priceSelector, &bestPrice_r, chromedp.ByQuery),
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
	projected := int(info[7].(float64))

	log.Println("Comparing", bestPrice_r, "to", "RAP", RAP, "and value", value)
	if !BuyCheck(bestPrice, RAP, value) || projected != -1 { //Failed price validation!
		log.Println("Failed price validation! Canceling..")
		return false
	}

	//Click purchase
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(*buySelector, chromedp.ByQuery),
		chromedp.Click(*buySelector, chromedp.NodeVisible, chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	log.Println("Clicked buy button")

	// Wait for purchase modal to appear (a bit of extra time for network)
	time.Sleep(1000 * time.Millisecond)

	err = chromedp.Run(ctx, //Confirm click
		chromedp.WaitVisible(*confirmSelector, chromedp.ByQuery),
		chromedp.Sleep(1000*time.Millisecond),
		chromedp.Click(*confirmSelector, chromedp.NodeVisible, chromedp.ByQuery),
	)

	if err != nil {
		log.Printf("Error: %v\n", err)
		return false
	}

	log.Println("Executed buy order on:", name, id)

	time.Sleep(5 * time.Second)

	return true
}
