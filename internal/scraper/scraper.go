package scraper

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb"
)

// Scraper orchestrates per-site scraping strategies. Site-specific logic lives
// behind the SiteScraper interface (see village.go and viafoura.go); this type
// owns the worker pools, scheduling, and storage.
type Scraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	pool    *BrowserPool
	config  *ScrapingConfig

	strategies map[internal.SiteType]SiteScraper
}

// NewScraper creates a new scraper with configuration
func NewScraper(ctx context.Context, config *ScrapingConfig) (*Scraper, error) {
	if config == nil {
		config = DefaultConfig()
	}

	logEntry := logger.New(ctx)

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, &ScrapingError{Op: "InitializePlaywright", Err: err.Error()}
	}

	// Launch browser with optimized settings
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(config.HeadlessMode),
		Args:     config.BrowserArgs(),
	})
	if err != nil {
		pw.Stop()
		return nil, &ScrapingError{Op: "LaunchBrowser", Err: err.Error()}
	}

	// Create browser pool for efficient context reuse
	pool := NewBrowserPool(ctx, browser, config)

	httpClient := &http.Client{Timeout: 30 * time.Second}

	s := &Scraper{
		pw:      pw,
		browser: browser,
		pool:    pool,
		config:  config,
	}
	s.strategies = map[internal.SiteType]SiteScraper{
		internal.SiteTypeVillage:  newVillageScraper(pool, config),
		internal.SiteTypeViafoura: newViafouraScraper(httpClient, config),
	}

	logEntry.WithFields(logrus.Fields{
		"article_workers": config.MaxArticleWorkers,
		"comment_workers": config.MaxCommentWorkers,
		"headless":        config.HeadlessMode,
	}).Info("Scraper initialized")

	return s, nil
}

// Close properly shuts down the scraper
func (s *Scraper) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		return s.pw.Stop()
	}
	return nil
}

// strategyForSite resolves the scraping strategy for a site config.
func (s *Scraper) strategyForSite(cfg internal.SiteConfig) SiteScraper {
	return s.strategies[cfg.Type]
}

// strategyForArticle resolves the strategy for an already-discovered article.
// Articles cross-posted across Village sites carry AllSitesName; those (and any
// unknown site) fall back to the Village strategy.
func (s *Scraper) strategyForArticle(article *store.Article) SiteScraper {
	if cfg, ok := internal.SitesMap[article.SiteName]; ok {
		return s.strategies[cfg.Type]
	}
	return s.strategies[internal.SiteTypeVillage]
}

// ScrapeAndStoreArticles scrapes articles from all configured sites
func (s *Scraper) ScrapeAndStoreArticles(ctx context.Context) error {
	logEntry := logger.New(ctx).WithField("operation", "scrape_articles")

	storage, err := rdb.New(ctx)
	if err != nil {
		return &ScrapingError{Op: "CreateStorage", Err: err.Error()}
	}

	articlesMap, err := s.scrapeArticlesConcurrently(ctx, internal.SitesMap)
	if err != nil {
		return err
	}

	if len(articlesMap) == 0 {
		logEntry.Info("No articles found to store")
		return nil
	}

	articlesToAdd := make([]*store.Article, 0, len(articlesMap))
	for _, article := range articlesMap {
		articlesToAdd = append(articlesToAdd, article)
	}

	if err := storage.AddArticles(ctx, articlesToAdd...); err != nil {
		return &ScrapingError{Op: "StoreArticles", Err: err.Error()}
	}

	logEntry.WithField("articles_stored", len(articlesToAdd)).Info("Articles scraping completed")
	return nil
}

// ScrapeAndStoreComments scrapes comments for recent articles
func (s *Scraper) ScrapeAndStoreComments(ctx context.Context, daysAgo int, forceScrape bool) error {
	logEntry := logger.New(ctx).WithField("operation", "scrape_comments")

	storage, err := rdb.New(ctx)
	if err != nil {
		return &ScrapingError{Op: "CreateStorage", Err: err.Error()}
	}

	// Get articles to process
	articles, err := s.getArticlesToScrape(ctx, storage, daysAgo, forceScrape)
	if err != nil {
		return err
	}

	if len(articles) == 0 {
		logEntry.Info("No articles need comment scraping")
		return nil
	}

	// Scrape comments concurrently
	comments, users, err := s.scrapeCommentsConcurrently(ctx, articles)
	if err != nil {
		return err
	}

	// Store results
	if err := s.storeCommentsAndUsers(ctx, storage, comments, users, articles); err != nil {
		return err
	}

	logEntry.WithFields(logrus.Fields{
		"articles_processed": len(articles),
		"comments_found":     len(comments),
		"users_found":        len(users),
	}).Info("Comments scraping completed")

	return nil
}

// scrapeArticlesConcurrently scrapes multiple sites concurrently
func (s *Scraper) scrapeArticlesConcurrently(ctx context.Context, sites map[string]internal.SiteConfig) (map[string]*store.Article, error) {
	logEntry := logger.New(ctx).WithField("operation", "concurrent_articles")

	siteChan := make(chan internal.SiteConfig, len(sites))
	for _, cfg := range sites {
		siteChan <- cfg
	}
	close(siteChan)

	var (
		allArticles = make(map[string]*store.Article)
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	logEntry.WithFields(logrus.Fields{
		"total_sites": len(sites),
		"workers":     s.config.MaxArticleWorkers,
	}).Info("Starting concurrent article scraping")

	// Start workers
	for workerID := range s.config.MaxArticleWorkers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.articleWorker(ctx, id, siteChan, &allArticles, &mu)
		}(workerID)
	}

	wg.Wait()

	logEntry.WithField("articles_found", len(allArticles)).Info("Article scraping completed")
	return allArticles, nil
}

// articleWorker processes sites for article scraping
func (s *Scraper) articleWorker(ctx context.Context, workerID int, siteChan <-chan internal.SiteConfig, results *map[string]*store.Article, mu *sync.Mutex) {
	workerLogger := logger.New(ctx).WithField("worker_id", workerID)

	for site := range siteChan {
		startTime := time.Now()

		strategy := s.strategyForSite(site)
		if strategy == nil {
			workerLogger.WithField("site", site.Name).Error("No scraping strategy for site type")
			continue
		}

		articles, err := strategy.DiscoverArticles(ctx, site)
		if err != nil {
			workerLogger.WithError(err).WithField("site", site.Name).Error("Failed to scrape site")
			continue
		}

		// Thread-safe update. The same article ID appearing under two different
		// site names means it's cross-posted across the Village network.
		mu.Lock()
		for _, article := range articles {
			if existing, exists := (*results)[article.ID]; exists {
				if existing.SiteName != article.SiteName {
					existing.SiteName = internal.AllSitesName
				}
			} else {
				(*results)[article.ID] = article
			}
		}
		mu.Unlock()

		workerLogger.WithFields(logrus.Fields{
			"site":     site.Name,
			"articles": len(articles),
			"duration": time.Since(startTime),
		}).Info("Site scraping completed")
	}
}

func (s *Scraper) getArticlesToScrape(ctx context.Context, storage store.Storage, daysAgo int, forceScrape bool) ([]*store.Article, error) {
	startingTime := time.Now().Add(-time.Hour * 24 * time.Duration(daysAgo))

	articlesSince, err := storage.GetRecentlyDiscoveredArticles(ctx, startingTime)
	if err != nil {
		return nil, &ScrapingError{Op: "GetRecentArticles", Err: err.Error()}
	}

	if forceScrape {
		return articlesSince, nil
	}

	// Filter articles based on scraping schedule
	now := time.Now()
	filtered := make([]*store.Article, 0, len(articlesSince))

	for _, article := range articlesSince {
		if s.shouldScrapeArticle(article, now) {
			filtered = append(filtered, article)
		}
	}

	return filtered, nil
}

func (s *Scraper) shouldScrapeArticle(article *store.Article, now time.Time) bool {
	hoursSinceDiscovery := now.Sub(article.DiscoveryTime).Hours()
	minsSinceLastScrape := now.Sub(article.LastScrapeTime).Minutes()

	switch {
	case hoursSinceDiscovery < 24:
		return minsSinceLastScrape > 15
	case hoursSinceDiscovery < 48:
		return minsSinceLastScrape > 30
	case hoursSinceDiscovery < 96:
		return minsSinceLastScrape > 60
	case hoursSinceDiscovery < 120:
		return minsSinceLastScrape > 120
	default:
		return minsSinceLastScrape > 240
	}
}

func (s *Scraper) scrapeCommentsConcurrently(ctx context.Context, articles []*store.Article) ([]*store.Comment, []*store.User, error) {
	logEntry := logger.New(ctx).WithField("operation", "concurrent_comments")

	articleChan := make(chan *store.Article, len(articles))
	for _, article := range articles {
		articleChan <- article
	}
	close(articleChan)

	var (
		allComments     []*store.Comment
		userIDToNameMap = make(map[string]string)
		mu              sync.Mutex
		wg              sync.WaitGroup
	)

	logEntry.WithFields(logrus.Fields{
		"total_articles": len(articles),
		"workers":        s.config.MaxCommentWorkers,
	}).Info("Starting concurrent comment scraping")

	// Start workers
	for workerID := range s.config.MaxCommentWorkers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.commentWorker(ctx, id, articleChan, &allComments, &userIDToNameMap, &mu)
		}(workerID)
	}

	wg.Wait()

	// Convert user map to slice
	users := make([]*store.User, 0, len(userIDToNameMap))
	for userID, userName := range userIDToNameMap {
		users = append(users, &store.User{
			ID:       userID,
			UserName: userName,
		})
	}

	logEntry.WithFields(logrus.Fields{
		"comments_found": len(allComments),
		"users_found":    len(users),
	}).Info("Comment scraping completed")

	return allComments, users, nil
}

// commentWorker processes articles for comment scraping
func (s *Scraper) commentWorker(ctx context.Context, workerID int, articleChan <-chan *store.Article, allComments *[]*store.Comment, userIDToNameMap *map[string]string, mu *sync.Mutex) {
	workerLogger := logger.New(ctx).WithField("worker_id", workerID)

	for article := range articleChan {
		startTime := time.Now()

		comments, users, err := s.ScrapeCommentsFromArticle(ctx, article)
		if err != nil {
			workerLogger.WithError(err).WithField("article_id", article.ID).Error("Failed to scrape comments")
			continue
		}

		// Thread-safe update
		mu.Lock()
		*allComments = append(*allComments, comments...)
		for _, user := range users {
			(*userIDToNameMap)[user.ID] = user.UserName
		}
		mu.Unlock()

		workerLogger.WithFields(logrus.Fields{
			"article_id":     article.ID,
			"comments_found": len(comments),
			"duration":       time.Since(startTime),
		}).Info("Article comment scraping completed")
	}
}

// ScrapeCommentsFromArticle scrapes comments + users for a single article by
// dispatching to the appropriate site strategy.
func (s *Scraper) ScrapeCommentsFromArticle(ctx context.Context, article *store.Article) ([]*store.Comment, []*store.User, error) {
	strategy := s.strategyForArticle(article)
	if strategy == nil {
		return nil, nil, &ScrapingError{Op: "ResolveStrategy", URL: article.Url, Err: "no scraping strategy for article"}
	}
	return strategy.ScrapeComments(ctx, article)
}

func (s *Scraper) storeCommentsAndUsers(ctx context.Context, storage store.Storage, comments []*store.Comment, users []*store.User, articles []*store.Article) error {
	// Store comments and users, update article scrape times
	if len(comments) > 0 {
		if err := storage.AddComments(ctx, comments); err != nil {
			return &ScrapingError{Op: "StoreComments", Err: err.Error()}
		}
	}

	if len(users) > 0 {
		if err := storage.AddUsers(ctx, users...); err != nil {
			return &ScrapingError{Op: "StoreUsers", Err: err.Error()}
		}
	}

	// Update scrape times
	articleIDs := make([]string, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	storage.SetArticleScrapedAt(ctx, time.Now(), articleIDs...)

	return nil
}
