package scraper

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb"
)

// PlaywrightScraper handles all scraping operations using Playwright
type PlaywrightScraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

// NewPlaywrightScraper creates a new Playwright-based scraper
func NewPlaywrightScraper(ctx context.Context) (*PlaywrightScraper, error) {
	logEntry := logger.New(ctx)

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		logEntry.WithError(err).Error("Failed to start Playwright")
		return nil, err
	}

	// Launch browser with stealth configurations
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Run headless for performance
		Args: []string{
			"--no-sandbox",
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-first-run",
			"--disable-extensions",
			"--disable-plugins",
			"--disable-images", // Faster loading by not loading images
			"--disable-background-timer-throttling",
			"--disable-backgrounding-occluded-windows",
			"--disable-renderer-backgrounding",
			"--disable-features=TranslateUI",
			"--disable-ipc-flooding-protection",
		},
	})
	if err != nil {
		logEntry.WithError(err).Error("Failed to launch browser")
		pw.Stop()
		return nil, err
	}

	return &PlaywrightScraper{
		pw:      pw,
		browser: browser,
	}, nil
}

// Close properly shuts down the Playwright scraper
func (ps *PlaywrightScraper) Close() error {
	if ps.browser != nil {
		ps.browser.Close()
	}
	if ps.pw != nil {
		return ps.pw.Stop()
	}
	return nil
}

// ScrapeArticles scrapes articles from a given site URL using Playwright
func (ps *PlaywrightScraper) ScrapeArticles(ctx context.Context, siteUrl string) map[int]*store.Article {
	logEntry := logger.New(ctx).WithField("site_url", siteUrl)

	// Create a new context with realistic browser settings
	context, err := ps.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(getRandomUserAgent()),
		Viewport:  &playwright.Size{Width: 1920, Height: 1080},
		ExtraHttpHeaders: map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"Accept-Language":           "en-US,en;q=0.9,en-CA;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "cross-site",
			"Sec-Fetch-User":            "?1",
			"Cache-Control":             "max-age=0",
			"Pragma":                    "no-cache",
		},
	})
	if err != nil {
		logEntry.WithError(err).Error("Failed to create browser context")
		return make(map[int]*store.Article)
	}
	defer context.Close()

	// Create a new page
	page, err := context.NewPage()
	if err != nil {
		logEntry.WithError(err).Error("Failed to create page")
		return make(map[int]*store.Article)
	}
	defer page.Close()

	// Add stealth modifications to avoid detection
	err = page.AddInitScript(playwright.Script{
		Content: playwright.String(`
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
				get: () => [1, 2, 3, 4, 5], // fake plugins
			});
			
			// Mock permissions
			const originalQuery = window.navigator.permissions.query;
			window.navigator.permissions.query = (parameters) => (
				parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters)
			);
			
			// Hide automation indicators
			window.chrome = {
				runtime: {},
			};
			
			// Mock screen properties
			Object.defineProperty(screen, 'colorDepth', { get: () => 24 });
			Object.defineProperty(screen, 'pixelDepth', { get: () => 24 });
		`),
	})
	if err != nil {
		logEntry.WithError(err).Warn("Failed to add stealth script")
	}

	// Add random delay before navigation to mimic human behavior
	randomDelay := time.Duration(500+time.Now().UnixNano()%1500) * time.Millisecond
	time.Sleep(randomDelay)

	// Navigate to the site with a timeout
	logEntry.Info("Navigating to site with Playwright")
	_, err = page.Goto(siteUrl, playwright.PageGotoOptions{
		Timeout:   playwright.Float(45000), // 45 second timeout
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		logEntry.WithError(err).Error("Failed to navigate to site")
		return make(map[int]*store.Article)
	}

	// Wait for the page to be fully loaded and add another delay
	time.Sleep(time.Duration(1000+time.Now().UnixNano()%2000) * time.Millisecond)

	// Get the page content
	content, err := page.Content()
	if err != nil {
		logEntry.WithError(err).Error("Failed to get page content")
		return make(map[int]*store.Article)
	}

	// Parse the content with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		logEntry.WithError(err).Error("Error parsing page content")
		return make(map[int]*store.Article)
	}

	// Use the same parsing logic as the original scraper
	articlesMap := make(map[int]*store.Article)
	doc.Find("a.section-item").Each(func(i int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists {
			if isArticleUrl(href) {
				title := sel.Find("div.section-title").First().Text()
				// Clean up titles
				title = strings.TrimSpace(title)
				title = strings.TrimRightFunc(title, func(r rune) bool {
					return unicode.IsNumber(r) || unicode.IsSpace(r)
				})
				if strings.HasPrefix(href, "http") {
					// Skip external URLs
					return
				}
				articleId := getArticleId(ctx, href)
				if _, ok := articlesMap[articleId]; !ok {
					articlesMap[articleId] = &store.Article{
						ID:             getArticleId(ctx, href),
						Title:          title,
						Url:            siteUrl + href,
						DiscoveryTime:  time.Now(),
						LastScrapeTime: time.Now(),
					}
				}
			}
		}
	})

	articleIDs := make([]int, 0, len(articlesMap))
	for id := range articlesMap {
		articleIDs = append(articleIDs, id)
	}
	logEntry.WithField("articles", articleIDs).Info("Articles found with Playwright")
	return articlesMap
}

// ScrapeAndStoreArticlesWithPlaywright is the main function that scrapes all sites using Playwright
func ScrapeAndStoreArticlesWithPlaywright(ctx context.Context) {
	logEntry := logger.New(ctx).WithField("job", "scraper/articles")
	storage, err := rdb.New(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Create the Playwright scraper
	scraper, err := NewPlaywrightScraper(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create Playwright scraper")
	}
	defer scraper.Close()

	articlesMap := make(map[int]*store.Article)
	totalSites := len(internal.SitesMap)
	currentSite := 0

	for siteName, siteUrl := range internal.SitesMap {
		currentSite++
		logEntry.WithFields(logrus.Fields{
			"progress": fmt.Sprintf("%d/%d", currentSite, totalSites),
			"site":     siteName,
		}).Info("Starting to scrape site")

		foundArticles := scraper.ScrapeArticles(ctx, siteUrl)

		for id, foundArticle := range foundArticles {
			if _, ok := articlesMap[id]; ok {
				foundArticle.SiteName = internal.AllSitesName
			} else {
				foundArticle.SiteName = siteName
			}
			articlesMap[foundArticle.ID] = foundArticle
		}

		logEntry.WithFields(logrus.Fields{
			"progress": fmt.Sprintf("%d/%d", currentSite, totalSites),
			"site":     siteName,
		}).Info("Completed scraping site")

		// Add delay between sites to avoid overwhelming servers
		if currentSite < totalSites {
			delay := time.Duration(2000+time.Now().UnixNano()%3000) * time.Millisecond
			time.Sleep(delay)
		}
	}

	articlesToAdd := make([]*store.Article, 0, len(articlesMap))
	for _, article := range articlesMap {
		articlesToAdd = append(articlesToAdd, article)
	}

	if len(articlesToAdd) == 0 {
		logEntry.Info("No articles to add, skipping")
	} else {
		err = storage.AddArticles(ctx, articlesToAdd...)
		if err != nil {
			logEntry.WithError(err).Error("failed to add articles")
		}
	}

	articleIDs := make([]int, 0, len(articlesMap))
	for id := range articlesMap {
		articleIDs = append(articleIDs, id)
	}
	logEntry.WithField("articles", articleIDs).Info("Found and stored articles")
}

// ScrapeAndStoreCommentsWithPlaywright scrapes comments using Playwright
func ScrapeAndStoreCommentsWithPlaywright(ctx context.Context, daysAgo int, forceScrape bool) {
	logEntry := logger.New(ctx).WithField("job", "scraper/comments")
	storage, err := rdb.New(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	startingTime := time.Now().Add(-time.Hour * 24 * time.Duration(daysAgo))

	// Get articles from X days ago
	articlesSince, err := storage.GetRecentlyDiscoveredArticles(ctx, startingTime)
	if err != nil {
		logEntry.WithError(err).Error("failed to get article ids")
		return
	}

	logEntry.WithFields(logrus.Fields{
		"article starting time": startingTime,
		"articles since":        len(articlesSince),
	}).Info("Fetching articles from storage to scrape")

	// determine if article should be scraped
	now := time.Now()
	articles := make([]*store.Article, 0)
	for _, article := range articlesSince {
		hoursSinceDiscovery := now.Sub(article.DiscoveryTime).Hours()
		minsSinceLastScrape := now.Sub(article.LastScrapeTime).Minutes()
		if forceScrape {
			articles = append(articles, article)
		} else if hoursSinceDiscovery < 24 && minsSinceLastScrape > 15 {
			articles = append(articles, article)
		} else if hoursSinceDiscovery < 48 && minsSinceLastScrape > 30 {
			articles = append(articles, article)
		} else if hoursSinceDiscovery < 96 && minsSinceLastScrape > 60 {
			articles = append(articles, article)
		} else if hoursSinceDiscovery < 120 && minsSinceLastScrape > 120 {
			articles = append(articles, article)
		} else if hoursSinceDiscovery >= 120 && minsSinceLastScrape > 240 {
			articles = append(articles, article)
		}
	}

	logEntry.WithField("articles", len(articles)).Info("Filtered articles to scrape")

	// Create the Playwright scraper for comments
	scraper, err := NewPlaywrightScraper(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create Playwright scraper")
	}
	defer scraper.Close()

	logEntry.WithField("job", "scraper/comments").Info("Starting to scrape comments with Playwright")
	comments, users := scraper.ScrapeCommentsFromArticles(ctx, articles)
	logEntry.WithFields(logrus.Fields{
		"job":      "scraper/comments",
		"comments": len(comments),
		"users":    len(users),
	}).Info("Found comments")

	logEntry.WithField("users", len(users)).Info("Found users")

	if err := storage.AddComments(ctx, comments); err != nil {
		logEntry.WithError(err).Error("failed to add comments ids")
	}
	logEntry.WithField("comments", len(comments)).Info("Added comments")

	if err := storage.AddUsers(ctx, users...); err != nil {
		logEntry.WithError(err).Error("failed to add user ids")
	}
	logEntry.Info("Added users")

	articleIDs := make([]int, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	storage.SetArticleScrapedAt(ctx, time.Now(), articleIDs...)
}

// ScrapeCommentsFromArticles scrapes comments from articles using Playwright
func (ps *PlaywrightScraper) ScrapeCommentsFromArticles(ctx context.Context, articles []*store.Article) ([]*store.Comment, []*store.User) {
	logEntry := logger.New(ctx)

	var comments []*store.Comment
	userIDToNameMap := make(map[int]string)
	totalArticles := len(articles)

	for i, article := range articles {
		currentProgress := i + 1
		logEntry.WithFields(logrus.Fields{
			"progress":   fmt.Sprintf("%d/%d", currentProgress, totalArticles),
			"articleId":  article.ID,
			"articleUrl": article.Url,
		}).Info("Starting to scrape comments for article")

		articleComments, err := ps.getCommentsFromArticle(ctx, article, userIDToNameMap)
		if err != nil {
			logEntry.WithError(err).WithFields(logrus.Fields{
				"progress":  fmt.Sprintf("%d/%d", currentProgress, totalArticles),
				"articleId": article.ID,
			}).Error("failed to get comments")
			continue
		}

		comments = append(comments, articleComments...)

		logEntry.WithFields(logrus.Fields{
			"progress":      fmt.Sprintf("%d/%d", currentProgress, totalArticles),
			"articleId":     article.ID,
			"commentsFound": len(articleComments),
			"totalComments": len(comments),
		}).Info("Completed scraping comments for article")
	}

	logEntry.WithFields(logrus.Fields{
		"articles": len(articles),
		"comments": len(comments),
		"users":    len(userIDToNameMap),
	}).Info("Completed scraping comments from all articles")

	users := make([]*store.User, 0, len(userIDToNameMap))
	for userID, userName := range userIDToNameMap {
		users = append(users, &store.User{
			ID:       userID,
			UserName: userName,
		})
	}

	return comments, users
}

// getCommentsFromArticle scrapes comments from a single article using Playwright
func (ps *PlaywrightScraper) getCommentsFromArticle(ctx context.Context, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, error) {
	logEntry := logger.New(ctx).WithField("articleId", article.ID)
	baseUrl, err := getBaseUrl(article.Url)
	if err != nil {
		logEntry.WithError(err).Error("failed to get host from url, cannot scrape this article!")
		return nil, err
	}

	var comments []*store.Comment
	var foundComments []*store.Comment
	loadMore := true
	lastParentId := 0

	pagesLeft := 10 // number to prevent infinite loop
	pageNum := 1
	for ; loadMore && pagesLeft >= 0; pagesLeft -= 1 {
		commentsUrl := fmt.Sprintf("%s/comments/get?Type=Comment&ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest", baseUrl, article.ID)
		if len(comments) > 0 {
			if lastParentId == 0 {
				// This should mean there aren't more comments
				break
			}
			commentsUrl += fmt.Sprintf("&lastId=%d", lastParentId)
		}
		logEntry.WithFields(logrus.Fields{
			"commentsUrl":   commentsUrl,
			"page":          pageNum,
			"commentsFound": len(comments),
		}).Info("Fetching comments")

		// Use Playwright to fetch comments
		content, err := ps.fetchPageContent(ctx, commentsUrl)
		if err != nil {
			logEntry.WithError(err).Error("failed to fetch comments with Playwright")
			continue
		}

		// Parse the content with goquery
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			logEntry.WithError(err).Error("Error parsing comments response")
			continue
		}

		foundComments, lastParentId = ps.searchArticleForCommentsWithPlaywright(ctx, doc, article, userIDToNameMap)
		comments = append(comments, foundComments...)

		if lastParentId == 0 {
			loadMore = false
		}
		pageNum++
	}

	if pagesLeft < 0 {
		logEntry.Error("Exceeded max pages")
	}

	return comments, nil
}

// fetchPageContent uses Playwright to fetch page content with stealth features
func (ps *PlaywrightScraper) fetchPageContent(ctx context.Context, url string) (string, error) {
	logEntry := logger.New(ctx).WithField("url", url)

	// Create a new context for this request
	context, err := ps.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(getRandomUserAgent()),
		Viewport:  &playwright.Size{Width: 1920, Height: 1080},
		ExtraHttpHeaders: map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"Accept-Language":           "en-US,en;q=0.9,en-CA;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "cross-site",
			"Sec-Fetch-User":            "?1",
			"Cache-Control":             "max-age=0",
			"Pragma":                    "no-cache",
		},
	})
	if err != nil {
		return "", err
	}
	defer context.Close()

	// Create a new page
	page, err := context.NewPage()
	if err != nil {
		return "", err
	}
	defer page.Close()

	// Add stealth script
	err = page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en', 'en-CA'],
			});
		`),
	})
	if err != nil {
		logEntry.WithError(err).Warn("Failed to add stealth script")
	}

	// Small delay before navigation
	randomDelay := time.Duration(200+time.Now().UnixNano()%800) * time.Millisecond
	time.Sleep(randomDelay)

	// Navigate to the URL
	_, err = page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30000), // 30 second timeout
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return "", err
	}

	// Wait a bit for dynamic content
	time.Sleep(time.Duration(500+time.Now().UnixNano()%1000) * time.Millisecond)

	// Get the page content
	content, err := page.Content()
	if err != nil {
		return "", err
	}

	return content, nil
}

// getRepliesWithPlaywright gets reply comments using Playwright
func (ps *PlaywrightScraper) getRepliesWithPlaywright(ctx context.Context, article *store.Article, parentID string, userIDToNameMap map[int]string) []*store.Comment {
	logEntry := logger.New(ctx).WithField("article_id", article.ID)
	baseUrl, _ := getBaseUrl(article.Url) // not worried about error because we would have errored on this in the parent function
	commentsUrl := fmt.Sprintf("%s/comments/get?ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest&lastId=%22%22&ParentId=%s", baseUrl, article.ID, parentID)

	content, err := ps.fetchPageContent(ctx, commentsUrl)
	if err != nil {
		logEntry.WithError(err).Error("Failed to fetch replies with Playwright")
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		logEntry.WithError(err).Error("Error parsing replies response")
		return nil
	}

	docComments := doc.Find("div.comment")
	var comments []*store.Comment
	docComments.Each(func(i int, reply *goquery.Selection) {
		comments = append(comments, newCommentFromDiv(ctx, reply, article.ID, userIDToNameMap))
	})
	return comments
}

// getRandomUserAgent returns a random user agent string to vary requests
func getRandomUserAgent() string {
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

// Playwright-specific helper functions to avoid 403s on reply requests

// searchArticleForCommentsWithPlaywright uses Playwright for reply fetching
func (ps *PlaywrightScraper) searchArticleForCommentsWithPlaywright(ctx context.Context, commentsDoc *goquery.Document, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, int) {
	commentDivs := commentsDoc.Find("div.comment")
	return ps.getCommentsWithPlaywright(ctx, commentDivs, article, userIDToNameMap)
}

// getCommentsWithPlaywright processes comments using Playwright for replies
func (ps *PlaywrightScraper) getCommentsWithPlaywright(ctx context.Context, commentDivs *goquery.Selection, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, int) {
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

				// get the replies using Playwright
				replies := ps.getRepliesWithPlaywright(ctx, article, strconv.Itoa(getCommentID(ctx, commentDiv)), userIDToNameMap)
				comments = append(comments, replies...)

			} else { // If the button exists, get all replies
				if numReplies > 20 {
					logEntry.Warn("HUGE REPLY CHAIN FOUND!")
				} else {
					// Use Playwright for replies
					replies := ps.getRepliesWithPlaywright(ctx, article, loadMore.AttrOr("data-parent", ""), userIDToNameMap)
					comments = append(comments, replies...)
				}
			}
		}
	})

	return comments, lastParentId
}
