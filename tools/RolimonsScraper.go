package tools

/*
Scrapes item details (best price) directly from Rolimon item page
*/

import (
	"context"
	"fmt"
	"robolimited/config"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// ChromeDB scraper for Rolimon's website
type RolimonsScraper struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	mu          sync.Mutex
	isAlive     bool
}

// Constructor
func NewRolimonsScraper() *RolimonsScraper {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.Flag("disable-images", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.WindowSize(800, 600),
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	return &RolimonsScraper{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		isAlive:     true,
	}
}

// Gets price of specific limited
func (rs *RolimonsScraper) getBestPrice(id string) (string, error) {
	rs.mu.Lock()
	alive := rs.isAlive
	allocCtx := rs.allocCtx
	rs.mu.Unlock()

	if !alive {
		return "", fmt.Errorf("scraper has been closed")
	}

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 15*time.Second)
	defer timeoutCancel()

	var bestPrice string

	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(fmt.Sprintf(config.RolimonsSite, id)),
		chromedp.WaitVisible(`.value-stats-grid`, chromedp.ByQuery),
		chromedp.Evaluate(`
            (() => {
                const boxes = document.querySelectorAll('.value-stat-box');
                for (const box of boxes) {
                    const header = box.querySelector('.value-stat-header');
                    if (header && header.textContent.includes('Best Price')) {
                        const data = box.querySelector('.value-stat-data');
                        return data ? data.textContent.trim() : '';
                    }
                }
                return '';
            })();
        `, &bestPrice),
	)

	if err != nil && err.Error() == "context canceled" {
		return "", fmt.Errorf("request timeout (15s exceeded)")
	}
	return bestPrice, err
}

func (rs *RolimonsScraper) Close() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.isAlive {
		rs.isAlive = false
		rs.allocCancel()
	}
}

type ScraperPool struct {
	scrapers []*RolimonsScraper
}

// NewScraperPool creates multiple Chrome instances for concurrent use
func NewScraperPool(poolSize int) *ScraperPool {
	scrapers := make([]*RolimonsScraper, poolSize)

	for i := 0; i < poolSize; i++ {
		scrapers[i] = NewRolimonsScraper()
	}

	return &ScraperPool{
		scrapers: scrapers,
	}
}

// IsAlive checks if the scraper is still usable
func (rs *RolimonsScraper) IsAlive() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.isAlive
}

// ProcessConcurrent processes items concurrently using the pool
func (sp *ScraperPool) ProcessConcurrent(itemIDs []string) map[string]string {
	var results sync.Map
	var mu sync.Mutex
	var wg sync.WaitGroup

	poolSize := len(sp.scrapers)
	batchSize := (len(itemIDs) + poolSize - 1) / poolSize // Ceiling division

	fmt.Printf("Processing %d items in batches of ~%d using %d Chrome instances...\n",
		len(itemIDs), batchSize, poolSize)

	// Split itemIDs into batches
	for i := 0; i < poolSize && i*batchSize < len(itemIDs); i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(itemIDs) {
			end = len(itemIDs)
		}
		batch := itemIDs[start:end]

		wg.Add(1)
		go func(scraperIndex int, itemBatch []string) {
			defer wg.Done()

			scraper := sp.scrapers[scraperIndex] // Use dedicated scraper for this batch

			for _, itemID := range itemBatch {
				price, err := scraper.getBestPrice(itemID)

				mu.Lock()
				if err != nil {
					results.Store(itemID, "ERROR: "+err.Error())
				} else {
					results.Store(itemID, price)
				}
				mu.Unlock()

				fmt.Printf("Batch %d completed: %s -> %s\n", scraperIndex, itemID, price)

				// Small delay between requests to avoid rate limiting
				time.Sleep(500 * time.Millisecond)
			}
		}(i, batch)
	}
	wg.Wait()

	// Convert sync.Map back to regular map
	finalResults := make(map[string]string)
	results.Range(func(key, value interface{}) bool {
		finalResults[key.(string)] = value.(string)
		return true
	})
	return finalResults
}

// Close all scrapers in the pool
func (sp *ScraperPool) Close() {
	for _, scraper := range sp.scrapers {
		scraper.Close()
	}
}
