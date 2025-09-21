package scraper

import (
	"context"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/salt-today/salttoday2/internal/logger"
)

// BrowserPool manages a pool of reusable browser contexts for performance
type BrowserPool struct {
	browser playwright.Browser
	config  *ScrapingConfig

	// Pool management
	contexts chan playwright.BrowserContext
	mu       sync.Mutex
	closed   bool
}

// NewBrowserPool creates a new browser pool with the given configuration
func NewBrowserPool(ctx context.Context, browser playwright.Browser, config *ScrapingConfig) *BrowserPool {
	pool := &BrowserPool{
		browser:  browser,
		config:   config,
		contexts: make(chan playwright.BrowserContext, config.MaxArticleWorkers+config.MaxCommentWorkers),
	}

	// Pre-populate the pool with some contexts
	initialSize := min(5, config.MaxArticleWorkers+config.MaxCommentWorkers)
	for range initialSize {
		if ctx, err := pool.createContext(); err == nil {
			pool.contexts <- ctx
		}
	}

	return pool
}

// GetContext gets a browser context from the pool or creates a new one
func (bp *BrowserPool) GetContext(ctx context.Context) (playwright.BrowserContext, error) {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		return nil, &ScrapingError{Op: "GetContext", Err: "browser pool is closed"}
	}
	bp.mu.Unlock()

	select {
	case browserCtx := <-bp.contexts:
		return browserCtx, nil
	default:
		// No context available, create a new one
		return bp.createContext()
	}
}

// ReturnContext returns a context to the pool for reuse
func (bp *BrowserPool) ReturnContext(browserCtx playwright.BrowserContext) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed {
		browserCtx.Close()
		return
	}

	select {
	case bp.contexts <- browserCtx:
		// Successfully returned to pool
	default:
		// Pool is full, close the context
		browserCtx.Close()
	}
}

// Close closes all browser contexts and the pool
func (bp *BrowserPool) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed {
		return
	}
	bp.closed = true

	close(bp.contexts)
	for browserCtx := range bp.contexts {
		browserCtx.Close()
	}
}

// createContext creates a new browser context with standard configuration
func (bp *BrowserPool) createContext() (playwright.BrowserContext, error) {
	return bp.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(getRandomUserAgentInternal()),
		Viewport:  &playwright.Size{Width: 1920, Height: 1080},
	})
}

// WithContext executes a function with a browser context from the pool
func (bp *BrowserPool) WithContext(ctx context.Context, fn func(playwright.BrowserContext) error) error {
	browserCtx, err := bp.GetContext(ctx)
	if err != nil {
		return err
	}
	defer bp.ReturnContext(browserCtx)

	return fn(browserCtx)
}

// WithPage executes a function with a page created from a pooled context
func (bp *BrowserPool) WithPage(ctx context.Context, fn func(playwright.Page) error) error {
	return bp.WithContext(ctx, func(browserCtx playwright.BrowserContext) error {
		page, err := browserCtx.NewPage()
		if err != nil {
			return &ScrapingError{Op: "CreatePage", Err: err.Error()}
		}
		defer page.Close()

		// Add stealth script
		if err := page.AddInitScript(playwright.Script{
			Content: playwright.String(bp.config.StealthScript()),
		}); err != nil {
			logger.New(ctx).WithError(err).Warn("Failed to add stealth script")
		}

		return fn(page)
	})
}

// ScrapingError represents an error that occurred during scraping
type ScrapingError struct {
	Op  string // Operation that failed
	URL string // URL being scraped (optional)
	Err string // Error message
}

func (e *ScrapingError) Error() string {
	if e.URL != "" {
		return e.Op + " failed for " + e.URL + ": " + e.Err
	}
	return e.Op + " failed: " + e.Err
}

// getRandomUserAgentInternal returns a random user agent string (internal version)
func getRandomUserAgentInternal() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
	}

	return userAgents[time.Now().Unix()%int64(len(userAgents))]
}
