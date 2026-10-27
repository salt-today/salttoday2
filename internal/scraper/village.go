package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mxschmitt/playwright-go"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/store"
)

// villageScraper scrapes Village Media properties: articles are listed on the
// homepage (a.section-item) and comments come from the /comments/get AJAX
// endpoint, both rendered via Playwright. Article/comment/user IDs are integers
// encoded in URLs; they are stored as their decimal string form.
type villageScraper struct {
	pool   *BrowserPool
	config *ScrapingConfig
}

func newVillageScraper(pool *BrowserPool, config *ScrapingConfig) *villageScraper {
	return &villageScraper{pool: pool, config: config}
}

func (v *villageScraper) DiscoverArticles(ctx context.Context, site internal.SiteConfig) ([]*store.Article, error) {
	var articles []*store.Article

	err := v.pool.WithPage(ctx, func(page playwright.Page) error {
		if err := v.gotoWithRetry(ctx, page, site.BaseURL, v.config.NavigationTimeout); err != nil {
			return &ScrapingError{Op: "Navigate", URL: site.BaseURL, Err: err.Error()}
		}

		content, err := page.Content()
		if err != nil {
			return &ScrapingError{Op: "GetContent", URL: site.BaseURL, Err: err.Error()}
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return &ScrapingError{Op: "ParseHTML", URL: site.BaseURL, Err: err.Error()}
		}

		articles = v.parseArticlesFromDoc(ctx, doc, site)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return articles, nil
}

// gotoWithRetry navigates to url, retrying with exponential backoff when the
// navigation fails (typically a timeout from a slow or throttled site).
func (v *villageScraper) gotoWithRetry(ctx context.Context, page playwright.Page, url string, timeout time.Duration) error {
	delay := v.config.RetryDelay
	var err error
	for attempt := 0; attempt <= v.config.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.New(ctx).WithError(err).WithField("url", url).WithField("attempt", attempt).Warn("Navigation failed, retrying")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * v.config.BackoffMultiple)
		}
		_, err = page.Goto(url, playwright.PageGotoOptions{
			Timeout:   playwright.Float(float64(timeout.Milliseconds())),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if err == nil {
			return nil
		}
	}
	return err
}

// parseArticlesFromDoc extracts articles from parsed homepage HTML.
func (v *villageScraper) parseArticlesFromDoc(ctx context.Context, doc *goquery.Document, site internal.SiteConfig) []*store.Article {
	var articles []*store.Article

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
		if articleID == "" {
			return // Skip invalid article IDs
		}

		articles = append(articles, &store.Article{
			ID:             articleID,
			Title:          title,
			SiteName:       site.Name,
			Url:            site.BaseURL + href,
			DiscoveryTime:  time.Now(),
			LastScrapeTime: time.Now(),
		})
	})

	return articles
}

func (v *villageScraper) ScrapeComments(ctx context.Context, article *store.Article) ([]*store.Comment, []*store.User, error) {
	baseUrl, err := getBaseUrl(article.Url)
	if err != nil {
		return nil, nil, &ScrapingError{Op: "GetBaseURL", URL: article.Url, Err: err.Error()}
	}

	userIDToNameMap := make(map[string]string)
	var allComments []*store.Comment
	lastParentId := 0
	pageNum := 1

	for pageNum <= v.config.MaxCommentPages && lastParentId >= 0 {
		commentsUrl := fmt.Sprintf("%s/comments/get?Type=Comment&ContentId=%s&TagId=2346&TagType=Content&Sort=Oldest", baseUrl, article.ID)
		if len(allComments) > 0 && lastParentId > 0 {
			commentsUrl += fmt.Sprintf("&lastId=%d", lastParentId)
		}

		comments, newLastParentId, err := v.scrapeCommentsFromPage(ctx, commentsUrl, article, userIDToNameMap)
		if err != nil {
			return nil, nil, err
		}

		allComments = append(allComments, comments...)
		lastParentId = newLastParentId

		if lastParentId == 0 {
			break // No more comments
		}
		pageNum++
	}

	return allComments, usersFromMap(userIDToNameMap), nil
}

// scrapeCommentsFromPage scrapes comments from a single page
func (v *villageScraper) scrapeCommentsFromPage(ctx context.Context, commentsUrl string, article *store.Article, userIDToNameMap map[string]string) ([]*store.Comment, int, error) {
	var comments []*store.Comment
	var lastParentId int

	err := v.pool.WithPage(ctx, func(page playwright.Page) error {
		if err := v.gotoWithRetry(ctx, page, commentsUrl, v.config.PageLoadTimeout); err != nil {
			return &ScrapingError{Op: "NavigateComments", URL: commentsUrl, Err: err.Error()}
		}

		content, err := page.Content()
		if err != nil {
			return &ScrapingError{Op: "GetCommentsContent", URL: commentsUrl, Err: err.Error()}
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return &ScrapingError{Op: "ParseCommentsHTML", URL: commentsUrl, Err: err.Error()}
		}

		comments, lastParentId = v.getCommentsEnhanced(ctx, doc.Find("div.comment"), article, userIDToNameMap)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	return comments, lastParentId, nil
}

// getCommentsEnhanced processes comments using enhanced scraper (no fatal errors, uses Playwright for replies)
func (v *villageScraper) getCommentsEnhanced(ctx context.Context, commentDivs *goquery.Selection, article *store.Article, userIDToNameMap map[string]string) ([]*store.Comment, int) {
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
			if id, err := strconv.Atoi(comment.ID); err == nil {
				lastParentId = id
			}
		} else { // If the comment has replies, check for a "Load More" button
			loadMore := commentDiv.Find("button.comments-more")

			// If the button doesn't exist, just get the comments
			if loadMore.Length() == 0 {
				// get the parent comment
				comments = append(comments, newCommentFromDiv(ctx, commentDiv, article.ID, userIDToNameMap))

				// get the replies using enhanced scraper (no fatal errors)
				replies := v.getRepliesEnhanced(ctx, article, getCommentID(ctx, commentDiv), userIDToNameMap)
				comments = append(comments, replies...)

			} else { // If the button exists, get all replies
				if numReplies > v.config.MaxReplyChain {
					logEntry.WithField("num_replies", numReplies).Warn("Large reply chain found, may be truncated")
				}
				// Use enhanced scraper for replies (no fatal errors)
				replies := v.getRepliesEnhanced(ctx, article, loadMore.AttrOr("data-parent", ""), userIDToNameMap)
				comments = append(comments, replies...)
			}
		}
	})

	return comments, lastParentId
}

// getRepliesEnhanced gets reply comments using enhanced scraper (no fatal errors, uses Playwright)
func (v *villageScraper) getRepliesEnhanced(ctx context.Context, article *store.Article, parentID string, userIDToNameMap map[string]string) []*store.Comment {
	logEntry := logger.New(ctx).WithField("article_id", article.ID)
	baseUrl, err := getBaseUrl(article.Url)
	if err != nil {
		logEntry.WithError(err).Error("Failed to get base URL for replies")
		return nil
	}

	commentsUrl := fmt.Sprintf("%s/comments/get?ContentId=%s&TagId=2346&TagType=Content&Sort=Oldest&lastId=%%22%%22&ParentId=%s", baseUrl, article.ID, parentID)

	var comments []*store.Comment
	err = v.pool.WithPage(ctx, func(page playwright.Page) error {
		if err := v.gotoWithRetry(ctx, page, commentsUrl, v.config.PageLoadTimeout); err != nil {
			return &ScrapingError{Op: "NavigateReplies", URL: commentsUrl, Err: err.Error()}
		}

		content, err := page.Content()
		if err != nil {
			return &ScrapingError{Op: "GetRepliesContent", URL: commentsUrl, Err: err.Error()}
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return &ScrapingError{Op: "ParseRepliesHTML", URL: commentsUrl, Err: err.Error()}
		}

		doc.Find("div.comment").Each(func(i int, reply *goquery.Selection) {
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

// nonArticleSections are top-level paths on Village sites that never hold
// commentable articles (ads, listings, account pages).
var nonArticleSections = map[string]bool{
	"classifieds": true,
	"directory":   true,
	"flyers":      true,
	"account":     true,
	"ads":         true,
	"submissions": true,
	"cssb":        true,
	"other":       true,
	"cares":       true,
}

// isArticleUrl reports whether href is an internal link outside the
// non-article sections. Whether it is really an article is decided by
// getArticleId, which requires a trailing numeric ID.
func isArticleUrl(href string) bool {
	if !strings.HasPrefix(href, "/") {
		return false
	}
	section, _, _ := strings.Cut(strings.TrimPrefix(href, "/"), "/")
	return !nonArticleSections[section]
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

func getCommentID(ctx context.Context, s *goquery.Selection) string {
	idString := s.AttrOr("data-id", "")
	if _, err := strconv.Atoi(idString); err != nil {
		logger.New(ctx).WithError(err).Error("Couldn't parse comment ID, using 0")
		return "0"
	}
	return idString
}

func getUserID(ctx context.Context, s *goquery.Selection) string {
	profileHref := s.Find("a.comment-un").AttrOr("href", "0")
	splitProfile := strings.Split(profileHref, "/")
	idString := splitProfile[len(splitProfile)-1]
	if _, err := strconv.Atoi(idString); err != nil {
		logger.New(ctx).WithError(err).Error("Couldn't parse user ID, using 0")
		return "0"
	}
	return idString
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

func newCommentFromDiv(ctx context.Context, div *goquery.Selection, articleID string, userIDToNameMap map[string]string) *store.Comment {
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

// getArticleId extracts the trailing integer ID from a Village article URL and
// returns it as a string. Returns "" if the URL has no numeric trailing ID.
func getArticleId(ctx context.Context, articleURL string) string {
	articleIdStr := articleURL[strings.LastIndex(articleURL, "-")+1:]
	if _, err := strconv.Atoi(articleIdStr); err != nil {
		logger.New(ctx).WithError(err).Error("Error converting article ID to int, skipping")
		return ""
	}
	return articleIdStr
}

// usersFromMap converts a userID->name map into a slice of users.
func usersFromMap(userIDToNameMap map[string]string) []*store.User {
	users := make([]*store.User, 0, len(userIDToNameMap))
	for id, name := range userIDToNameMap {
		users = append(users, &store.User{ID: id, UserName: name})
	}
	return users
}
