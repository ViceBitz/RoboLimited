package tools

import (
	"context"
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

	// Initialize with about:blank
	err := chromedp.Run(ctx, chromedp.Navigate("about:blank"))
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

// Reset navigates back to about:blank
func (b *Browser) Reset() error {
	return chromedp.Run(b.ctx, chromedp.Navigate("about:blank"))
}

// GetContext returns the chromedp context
func (b *Browser) GetContext() context.Context {
	return b.ctx
}

// GetContextWithTimeout returns the chromedp context with a timeout
// Always resets to about:blank when cancel() is called
func (b *Browser) GetContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	timeoutCtx, cancel := context.WithTimeout(b.ctx, timeout)

	wrappedCancel := func() {
		cancel()
		b.Reset() //Reset to about:blank
	}

	return timeoutCtx, wrappedCancel
}

// Close closes the browser (optional cleanup method)
func (b *Browser) Close() {
	b.cancel()
}
