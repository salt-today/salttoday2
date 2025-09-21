package scraper

import "time"

// ScrapingConfig holds all configuration for the scraper
type ScrapingConfig struct {
	// Concurrency settings
	MaxArticleWorkers int
	MaxCommentWorkers int

	// Timeout settings
	NavigationTimeout time.Duration
	PageLoadTimeout   time.Duration

	// Retry settings
	MaxRetries      int
	RetryDelay      time.Duration
	BackoffMultiple float64

	// Browser settings
	HeadlessMode  bool
	DisableImages bool

	// Limits
	MaxCommentPages int
	MaxReplyChain   int
}

// DefaultConfig returns sensible defaults for scraping
func DefaultConfig() *ScrapingConfig {
	return &ScrapingConfig{
		MaxArticleWorkers: 5,
		MaxCommentWorkers: 8,

		NavigationTimeout: 10 * time.Second,
		PageLoadTimeout:   8 * time.Second,

		MaxRetries:      3,
		RetryDelay:      1 * time.Second,
		BackoffMultiple: 2.0,

		HeadlessMode:  true,
		DisableImages: true,

		MaxCommentPages: 10,
		MaxReplyChain:   20,
	}
}

// BrowserArgs returns optimized browser launch arguments
func (c *ScrapingConfig) BrowserArgs() []string {
	args := []string{
		"--no-sandbox",
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--disable-extensions",
		"--disable-plugins",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--disable-features=TranslateUI",
		"--disable-ipc-flooding-protection",
	}

	if c.DisableImages {
		args = append(args, "--disable-images")
	}

	return args
}

// StealthScript returns JavaScript to avoid detection
func (c *ScrapingConfig) StealthScript() string {
	return `
		// Remove webdriver property
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
		});
		
		// Mock languages and plugins to look more realistic
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en', 'en-CA'],
		});
		
		// Override the plugins property to mimic a real browser
		Object.defineProperty(navigator, 'plugins', {
			get: () => [1, 2, 3, 4, 5],
		});
		
		// Hide automation indicators
		window.chrome = { runtime: {} };
		
		// Mock screen properties
		Object.defineProperty(screen, 'colorDepth', { get: () => 24 });
		Object.defineProperty(screen, 'pixelDepth', { get: () => 24 });
	`
}
