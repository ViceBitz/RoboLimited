package tools

/*
Stores list of ChromeDP flags for browser window tailored to different situations
*/

import (
	"robolimited/config"

	"github.com/chromedp/chromedp"
)

// Flag list for fastest page load possible
func FastFlags() []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// Core performance flags
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),

		// Network and loading optimizations
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-default-apps", true),

		// Memory and resource optimizations
		chromedp.Flag("memory-pressure-off", true),
		chromedp.Flag("max_old_space_size", "4096"),
		chromedp.Flag("aggressive-cache-discard", true),

		// Disable unnecessary features
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-images", true),      // Only if images not needed
		chromedp.Flag("disable-javascript", false), // Keep JS enabled

		// Audio/video optimizations
		chromedp.Flag("disable-audio-output", true),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),

		// Security features that can slow loading
		chromedp.Flag("disable-features", "TranslateUI,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-ipc-flooding-protection", true),

		// Chrome process optimizations
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-component-extensions-with-background-pages", true),

		// Logging and debugging (disable for speed)
		chromedp.Flag("disable-logging", true),
		chromedp.Flag("log-level", "3"), // Only fatal errors

		// User agent
		chromedp.Flag("user-agent", config.UserAgent),
	)

	return opts
}
