package tools

import (
	"context"
	"robolimited/config"
	"time"

	"github.com/chromedp/chromedp"
)

type Browser struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Constructor
func NewBrowser() (*Browser, error) {
	opts := FastFlags()

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)

	// Initialize with roblox homepage
	err := chromedp.Run(ctx, chromedp.Navigate(config.RobloxHome))
	if err != nil {
		cancel()
		allocCancel()
		return nil, err
	}

	return &Browser{
		ctx: ctx,
		cancel: func() {
			cancel()
			allocCancel()
		},
	}, nil
}

// Reset navigates back to homepage
func (b *Browser) Reset() error {
	return chromedp.Run(b.ctx, chromedp.Navigate(config.RobloxHome))
}

// GetContext returns the chromedp context
func (b *Browser) GetContext() context.Context {
	return b.ctx
}

// GetContextWithTimeout returns the chromedp context with a timeout
// Always resets to roblox homepage when cancel() is called
func (b *Browser) GetContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, timeout)

	wrappedCancel := func() {
		b.Reset() //Reset to roblox homepage
		cancel()
	}

	return timeoutCtx, wrappedCancel
}

// Close closes the browser (optional cleanup method)
func (b *Browser) Close() {
	b.cancel()
}
