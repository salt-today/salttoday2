package scraper

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb"
)

// Scraper provides high-performance web scraping with browser automation
type Scraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	pool    *BrowserPool
	config  *ScrapingConfig
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

	logEntry.WithFields(logrus.Fields{
		"article_workers": config.MaxArticleWorkers,
		"comment_workers": config.MaxCommentWorkers,
		"headless":        config.HeadlessMode,
	}).Info("Scraper initialized")

	return &Scraper{
		pw:      pw,
		browser: browser,
		pool:    pool,
		config:  config,
	}, nil
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
func (s *Scraper) scrapeArticlesConcurrently(ctx context.Context, sites map[string]string) (map[int]*store.Article, error) {
	logEntry := logger.New(ctx).WithField("operation", "concurrent_articles")

	siteChan := make(chan siteJob, len(sites))
	for name, url := range sites {
		siteChan <- siteJob{name: name, url: url}
	}
	close(siteChan)

	var (
		allArticles = make(map[int]*store.Article)
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

// siteJob represents a site to be scraped
type siteJob struct {
	name string
	url  string
}

// articleWorker processes sites for article scraping
func (s *Scraper) articleWorker(ctx context.Context, workerID int, siteChan <-chan siteJob, results *map[int]*store.Article, mu *sync.Mutex) {
	workerLogger := logger.New(ctx).WithField("worker_id", workerID)

	for site := range siteChan {
		startTime := time.Now()

		articles, err := s.scrapeArticlesFromSite(ctx, site.url)
		if err != nil {
			workerLogger.WithError(err).WithField("site", site.name).Error("Failed to scrape site")
			continue
		}

		// Thread-safe update
		mu.Lock()
		for id, article := range articles {
			if existing, exists := (*results)[id]; exists {
				existing.SiteName = internal.AllSitesName // Article appears on multiple sites
			} else {
				article.SiteName = site.name
				(*results)[id] = article
			}
		}
		mu.Unlock()

		workerLogger.WithFields(logrus.Fields{
			"site":     site.name,
			"articles": len(articles),
			"duration": time.Since(startTime),
		}).Info("Site scraping completed")
	}
}

// scrapeArticlesFromSite scrapes articles from a single site
func (s *Scraper) scrapeArticlesFromSite(ctx context.Context, siteURL string) (map[int]*store.Article, error) {
	var articles map[int]*store.Article

	err := s.pool.WithPage(ctx, func(page playwright.Page) error {
		// Navigate with timeout
		_, err := page.Goto(siteURL, playwright.PageGotoOptions{
			Timeout:   playwright.Float(float64(s.config.NavigationTimeout.Milliseconds())),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if err != nil {
			return &ScrapingError{Op: "Navigate", URL: siteURL, Err: err.Error()}
		}

		// Get page content
		content, err := page.Content()
		if err != nil {
			return &ScrapingError{Op: "GetContent", URL: siteURL, Err: err.Error()}
		}

		// Parse with goquery
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return &ScrapingError{Op: "ParseHTML", URL: siteURL, Err: err.Error()}
		}

		articles = s.parseArticlesFromDoc(ctx, doc, siteURL)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return articles, nil
}

// parseArticlesFromDoc extracts articles from parsed HTML
func (s *Scraper) parseArticlesFromDoc(ctx context.Context, doc *goquery.Document, siteURL string) map[int]*store.Article {
	articles := make(map[int]*store.Article)

	doc.Find("a.section-item").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || !isArticleUrl(href) || strings.HasPrefix(href, "http") {
			return // Skip external or invalid URLs
		}

		title := strings.TrimSpace(sel.Find("div.section-title").First().Text())
		title = strings.TrimRightFunc(title, func(r rune) bool {
			return (r >= '0' && r <= '9') || r == ' '
		})

		articleID := getArticleId(ctx, href)
		if articleID == 0 {
			return // Skip invalid article IDs
		}

		articles[articleID] = &store.Article{
			ID:             articleID,
			Title:          title,
			Url:            siteURL + href,
			DiscoveryTime:  time.Now(),
			LastScrapeTime: time.Now(),
		}
	})

	return articles
}

// Helper method stubs - implement remaining methods following the same patterns
func (s *Scraper) getArticlesToScrape(ctx context.Context, storage store.Storage, daysAgo int, forceScrape bool) ([]*store.Article, error) {
	// Implementation similar to existing logic but with better error handling
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
		userIDToNameMap = make(map[int]string)
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
func (s *Scraper) commentWorker(ctx context.Context, workerID int, articleChan <-chan *store.Article, allComments *[]*store.Comment, userIDToNameMap *map[int]string, mu *sync.Mutex) {
	workerLogger := logger.New(ctx).WithField("worker_id", workerID)

	for article := range articleChan {
		startTime := time.Now()

		localUserMap := make(map[int]string)
		comments, err := s.ScrapeCommentsFromArticle(ctx, article, localUserMap)
		if err != nil {
			workerLogger.WithError(err).WithField("article_id", article.ID).Error("Failed to scrape comments")
			continue
		}

		// Thread-safe update
		mu.Lock()
		*allComments = append(*allComments, comments...)
		for userID, userName := range localUserMap {
			(*userIDToNameMap)[userID] = userName
		}
		mu.Unlock()

		workerLogger.WithFields(logrus.Fields{
			"article_id":     article.ID,
			"comments_found": len(comments),
			"duration":       time.Since(startTime),
		}).Info("Article comment scraping completed")
	}
}

// ScrapeCommentsFromArticle scrapes comments from a single article (public method)
func (s *Scraper) ScrapeCommentsFromArticle(ctx context.Context, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, error) {
	baseUrl, err := getBaseUrl(article.Url)
	if err != nil {
		return nil, &ScrapingError{Op: "GetBaseURL", URL: article.Url, Err: err.Error()}
	}

	var allComments []*store.Comment
	lastParentId := 0
	pageNum := 1

	for pageNum <= s.config.MaxCommentPages && lastParentId >= 0 {
		commentsUrl := fmt.Sprintf("%s/comments/get?Type=Comment&ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest", baseUrl, article.ID)
		if len(allComments) > 0 && lastParentId > 0 {
			commentsUrl += fmt.Sprintf("&lastId=%d", lastParentId)
		}

		comments, newLastParentId, err := s.scrapeCommentsFromPage(ctx, commentsUrl, article, userIDToNameMap)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)
		lastParentId = newLastParentId

		if lastParentId == 0 {
			break // No more comments
		}
		pageNum++
	}

	return allComments, nil
}

// scrapeCommentsFromPage scrapes comments from a single page
func (s *Scraper) scrapeCommentsFromPage(ctx context.Context, commentsUrl string, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, int, error) {
	var comments []*store.Comment
	var lastParentId int

	err := s.pool.WithPage(ctx, func(page playwright.Page) error {
		// Navigate to comments URL
		_, err := page.Goto(commentsUrl, playwright.PageGotoOptions{
			Timeout:   playwright.Float(float64(s.config.PageLoadTimeout.Milliseconds())),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if err != nil {
			return &ScrapingError{Op: "NavigateComments", URL: commentsUrl, Err: err.Error()}
		}

		// Get page content
		content, err := page.Content()
		if err != nil {
			return &ScrapingError{Op: "GetCommentsContent", URL: commentsUrl, Err: err.Error()}
		}

		// Parse with goquery
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return &ScrapingError{Op: "ParseCommentsHTML", URL: commentsUrl, Err: err.Error()}
		}

		comments, lastParentId = s.parseCommentsFromDoc(ctx, doc, article, userIDToNameMap)
		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return comments, lastParentId, nil
}

// parseCommentsFromDoc extracts comments from parsed HTML using enhanced logic
func (s *Scraper) parseCommentsFromDoc(ctx context.Context, doc *goquery.Document, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, int) {
	commentDivs := doc.Find("div.comment")
	return s.getCommentsEnhanced(ctx, commentDivs, article, userIDToNameMap)
}

// getCommentsEnhanced processes comments using enhanced scraper (no fatal errors, uses Playwright for replies)
func (s *Scraper) getCommentsEnhanced(ctx context.Context, commentDivs *goquery.Selection, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, int) {
	logEntry := logger.New(ctx).WithField("article_id", article.ID)
	var comments []*store.Comment
	lastParentId := 0

	commentDivs.Each(func(i int, commentDiv *goquery.Selection) {
		numRepliesStr := commentDiv.AttrOr("data-replies", "0")
		numReplies, err := strconv.Atoi(numRepliesStr)
		if err != nil {
			logEntry.WithError(err).Error("Couldn't parse number of replies, assuming 0")
			numReplies = 0
		}

		if numReplies == 0 {
			// If comment has no replies, just get the comment itself
			comment := newCommentFromDiv(ctx, commentDiv, article.ID, userIDToNameMap)
			comments = append(comments, comment)
			lastParentId = comment.ID
		} else { // If the comment has replies, check for a "Load More" button
			loadMore := commentDiv.Find("button.comments-more")

			// If the button doesn't exist, just get the comments
			if loadMore.Length() == 0 {
				// get the parent comment
				comments = append(comments, newCommentFromDiv(ctx, commentDiv, article.ID, userIDToNameMap))

				// get the replies using enhanced scraper (no fatal errors)
				replies := s.getRepliesEnhanced(ctx, article, strconv.Itoa(getCommentID(ctx, commentDiv)), userIDToNameMap)
				comments = append(comments, replies...)

			} else { // If the button exists, get all replies
				if numReplies > s.config.MaxReplyChain {
					logEntry.WithField("num_replies", numReplies).Warn("Large reply chain found, may be truncated")
				}
				// Use enhanced scraper for replies (no fatal errors)
				replies := s.getRepliesEnhanced(ctx, article, loadMore.AttrOr("data-parent", ""), userIDToNameMap)
				comments = append(comments, replies...)
			}
		}
	})

	return comments, lastParentId
}

// getRepliesEnhanced gets reply comments using enhanced scraper (no fatal errors, uses Playwright)
func (s *Scraper) getRepliesEnhanced(ctx context.Context, article *store.Article, parentID string, userIDToNameMap map[int]string) []*store.Comment {
	logEntry := logger.New(ctx).WithField("article_id", article.ID)
	baseUrl, err := getBaseUrl(article.Url)
	if err != nil {
		logEntry.WithError(err).Error("Failed to get base URL for replies")
		return nil
	}

	commentsUrl := fmt.Sprintf("%s/comments/get?ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest&lastId=%22%22&ParentId=%s", baseUrl, article.ID, parentID)

	var comments []*store.Comment
	err = s.pool.WithPage(ctx, func(page playwright.Page) error {
		// Navigate to replies URL
		_, err := page.Goto(commentsUrl, playwright.PageGotoOptions{
			Timeout:   playwright.Float(float64(s.config.PageLoadTimeout.Milliseconds())),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if err != nil {
			return &ScrapingError{Op: "NavigateReplies", URL: commentsUrl, Err: err.Error()}
		}

		// Get page content
		content, err := page.Content()
		if err != nil {
			return &ScrapingError{Op: "GetRepliesContent", URL: commentsUrl, Err: err.Error()}
		}

		// Parse with goquery
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return &ScrapingError{Op: "ParseRepliesHTML", URL: commentsUrl, Err: err.Error()}
		}

		docComments := doc.Find("div.comment")
		docComments.Each(func(i int, reply *goquery.Selection) {
			comments = append(comments, newCommentFromDiv(ctx, reply, article.ID, userIDToNameMap))
		})
		return nil
	})

	if err != nil {
		logEntry.WithError(err).Error("Failed to fetch replies, skipping")
		return nil // Return empty slice instead of crashing
	}

	return comments
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
	articleIDs := make([]int, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	storage.SetArticleScrapedAt(ctx, time.Now(), articleIDs...)

	return nil
}

var articleCategories = []string{
	"/good-morning", // doesn't have a trailing slash because -thunder-bay, -sudbury, etc
	"/local-news/",
	"/columns/",
	"/local-business/",
	"/spotlight/",
	"/around-ontario/",
	"/great-stories/",
	"/videos/",
	"/opp-beat/",
	"/arts-culture/",
	"/local-sports/",
	"/closer-look/",
	"/local-entertainment",
	"/bulletin/",
	"/more-local/",
	"/city-police-beat/",
}

func isArticleUrl(href string) bool {
	for _, prefix := range articleCategories {
		if strings.Contains(href, prefix) {
			return true
		}
	}
	return false
}

func getBaseUrl(urlString string) (string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

func getContentHelper(s *goquery.Selection) string {
	return strings.TrimSpace(s.First().Text())
}

func getCommentID(ctx context.Context, s *goquery.Selection) int {
	logEntry := logger.New(ctx)
	idString := s.AttrOr("data-id", "")
	id, err := strconv.Atoi(idString)
	if err != nil {
		logEntry.WithError(err).Error("Couldn't parse comment ID, using 0")
		return 0
	}
	return id
}

func getUserID(ctx context.Context, s *goquery.Selection) int {
	profileHref := s.Find("a.comment-un").AttrOr("href", "0")
	splitProfile := strings.Split(profileHref, "/")
	idString := splitProfile[len(splitProfile)-1]
	id, err := strconv.Atoi(idString)
	if err != nil {
		logger.New(ctx).WithError(err).Error("Couldn't parse user ID, using 0")
		return 0
	}
	return id
}

func getUsername(_ context.Context, s *goquery.Selection) string {
	return getContentHelper(s.Find("a.comment-un"))
}

func getTimestamp(ctx context.Context, s *goquery.Selection) time.Time {
	now := time.Now()
	timeString := s.Find("time").AttrOr("datetime", now.String())
	commentTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		logger.New(ctx).WithError(err).Error("Couldn't parse time, using current time")
		return now
	}
	return commentTime
}

func getCommentText(_ context.Context, s *goquery.Selection) string {
	return getContentHelper(s.Find("div.comment-text"))
}

func getLikes(ctx context.Context, s *goquery.Selection) int32 {
	likeString := getContentHelper(s.Find("[value=Upvote]"))
	likes, err := strconv.Atoi(likeString)
	if err != nil {
		logger.New(ctx).WithError(err).Error("Unable to get likes")
		return 0
	}

	return int32(likes)
}

func getDislikes(ctx context.Context, s *goquery.Selection) int32 {
	dislikeString := getContentHelper(s.Find("[value=Downvote]"))
	dislikes, err := strconv.Atoi(dislikeString)
	if err != nil {
		logger.New(ctx).WithError(err).Error("Unable to get dislikes")
		return 0
	}

	return int32(dislikes)
}

func newCommentFromDiv(ctx context.Context, div *goquery.Selection, articleID int, userIDToNameMap map[int]string) *store.Comment {
	comment := &store.Comment{
		ID:       getCommentID(ctx, div),
		Article:  store.Article{ID: articleID},
		User:     store.User{ID: getUserID(ctx, div)},
		Time:     getTimestamp(ctx, div),
		Text:     getCommentText(ctx, div),
		Likes:    getLikes(ctx, div),
		Dislikes: getDislikes(ctx, div),
	}
	userIDToNameMap[comment.User.ID] = getUsername(ctx, div)
	return comment
}

func getNumberOfCommentApiCalls(ctx context.Context, doc *goquery.Document) int {
	topLevelCommentCountStr := doc.Find("div#comments").First().AttrOr("data-count", "0")
	topLevelCommentCount, err := strconv.Atoi(topLevelCommentCountStr)
	if err != nil {
		logger.New(ctx).WithError(err).Error("Error converting data-count to int")
		return 0
	}

	return int(math.Ceil(float64(topLevelCommentCount) / 20))
}

func getArticleId(ctx context.Context, url string) int {
	articleIdStr := url[strings.LastIndex(url, "-")+1:]
	articleId, err := strconv.Atoi(articleIdStr)
	if err != nil {
		logger.New(ctx).WithError(err).Error("Error converting article ID to int, using 0")
		return 0
	}
	return articleId
}
