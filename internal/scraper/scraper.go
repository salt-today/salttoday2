package scraper

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb"
)

// verify these are still correct and if there's any missing
var articleCategories = []string{
	"/good-morning", // doesn't have a trailing slash because -thunder-bay, -sudbury, etc
	"/local-news/",
	"/spotlight/",
	"/great-stories/",
	"/videos/",
	"/local-sports/",
	"/local-entertainment",
	"/bulletin/",
	"/more-local/",
	"/city-police-beat/",
}

func ScrapeAndStoreArticles(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("job", "scraper/articles")
	storage, err := rdb.New(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	articles := make([]*store.Article, 0)
	for _, site := range internal.GetSites() {
		foundArticles := ScrapeArticles(ctx, site)
		articles = append(articles, foundArticles...)
	}
	err = storage.AddArticles(ctx, articles...)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to add articles")
	}
	articleIDs := make([]int, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	logEntry.WithField("articles", articleIDs).Info("Found and stored articles")
}

func ScrapeAndStoreComments(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("job", "scraper/comments")
	storage, err := rdb.New(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Get articles from the last 7 days
	maxDays := 14
	articlesSince, err := storage.GetUnscrapedArticlesSince(ctx, time.Now().Add(-time.Hour*24*time.Duration(maxDays)))
	if err != nil {
		logEntry.WithError(err).Error("failed to get article ids")
		return
	}
	logEntry.WithField("articlesSince", articlesSince).Info("found articles since")

	// determine if article should be scraped
	// now := time.Now()
	articles := make([]*store.Article, 0)
	for _, article := range articlesSince {
		// for i := 0; i < maxDays; i++ {
		// 	multiplier := time.Duration(math.Pow(2, float64(i)))
		// 	if article.DiscoveryTime.After(now.Add(-time.Hour * 24 * time.Duration(i))) {
		// 		if article.LastScrapeTime.After(now.Add(-time.Minute * 15 * multiplier)) {
		articles = append(articles, article)
		//		}
		//		break
		//	}
		// }
	}
	logEntry.WithField("articlesToScrape", articles).Info("articlesToScrape")

	comments, users := ScrapeCommentsFromArticles(ctx, articles)
	commentIDs := make([]int, len(comments))
	for i, comment := range comments {
		commentIDs[i] = comment.ID
	}
	logEntry.WithField("comments", commentIDs).Info("Found comments")

	userIDs := make([]int, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}
	logEntry.WithField("users", userIDs).Info("Found users")

	if err := storage.AddComments(ctx, comments); err != nil {
		logEntry.WithError(err).Error("failed to add comments ids")
	}
	logEntry.WithField("comments", commentIDs).Info("Added comments")

	if err := storage.AddUsers(ctx, users...); err != nil {
		logEntry.WithError(err).Error("failed to add user ids")
	}
	logEntry.Info("Added users")

	// Update the last scrape time for the articles
	articleIDs := make([]int, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	storage.SetArticleScrapedAt(ctx, time.Now(), articleIDs...)

}

func ScrapeCommentsFromArticles(ctx context.Context, articles []*store.Article) ([]*store.Comment, []*store.User) {
	logEntry := sdk.Logger(ctx)

	comments := make([]*store.Comment, 0)
	userIDToNameMap := make(map[int]string)
	for _, article := range articles {
		articleComments, err := getCommentsFromArticle(ctx, article, userIDToNameMap)
		if err != nil {
			logEntry.WithError(err).Error("failed to get comments")
			continue
		}

		comments = append(comments, articleComments...)

		for _, comment := range comments {
			logEntry.WithFields(logrus.Fields{
				"comments found": comment,
				"article":        article,
			}).Info("Found comments from article")
		}
	}
	logEntry.WithFields(logrus.Fields{
		"articles": len(articles),
		"comments": len(comments),
	}).Info("Completed scraping site")

	users := make([]*store.User, 0)
	for userID, userName := range userIDToNameMap {
		users = append(users, &store.User{
			ID:       userID,
			UserName: userName,
		})
	}

	return comments, users
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

// This logic is ported from the old scraper so blame Tyler if it's bad
// The api for sootoday comments is awful so also thanks Tyler for figuring this out
func getCommentsFromArticle(ctx context.Context, article *store.Article, userIDToNameMap map[int]string) ([]*store.Comment, error) {
	logEntry := sdk.Logger(ctx).WithField("articleId", article.ID)
	baseUrl, err := getBaseUrl(article.Url)
	if err != nil {
		logEntry.WithError(err).Error("failed to get host from url, cannot scrape this article!")
	}

	// Loop at least once
	// Keep looping if theres a load more button
	comments := make([]*store.Comment, 0)
	loadMore := true
	for loadMore {
		commentsUrl := fmt.Sprintf("%s/comments/get?Type=Comment&ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest", baseUrl, article.ID)
		if len(comments) > 0 {
			commentsUrl += fmt.Sprintf("&lastId=%d", comments[len(comments)-1].ID)
		}

		res, err := http.Get(commentsUrl)
		if err != nil {
			logEntry.WithError(err).Error("failed to download comments")
			return nil, err
		}

		defer func() {
			if err := res.Body.Close(); err != nil {
				logEntry.WithError(err).Error("Error closing HTTP")
			}
		}()

		if res.StatusCode != 200 {
			logEntry.WithFields(logrus.Fields{
				"status_code": res.StatusCode,
				"status":      res.Status,
			}).Fatal("status code error")
			return nil, nil
		}

		initialCommentDoc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			logEntry.WithError(err).Error("Error loading HTTP response body")
			return nil, err
		}

		foundComments := searchArticleForComments(ctx, initialCommentDoc, article, userIDToNameMap)
		comments = append(comments, foundComments...)

		loadMoreSelection := initialCommentDoc.Find("button.comments-more")

		// If the button doesn't exist, just get the comments
		if loadMoreSelection.Length() == 0 {
			loadMore = false
		}
	}

	return comments, nil
}

func searchArticleForComments(ctx context.Context, commentsDoc *goquery.Document, article *store.Article, userIDToNameMap map[int]string) []*store.Comment {
	commentDivs := commentsDoc.Find("div.comment")
	comments := getComments(ctx, commentDivs, article, userIDToNameMap)

	return comments
}

func getComments(ctx context.Context, commentDivs *goquery.Selection, article *store.Article, userIDToNameMap map[int]string) []*store.Comment {
	logEntry := sdk.Logger(ctx).WithField("article_id", article.ID)
	comments := make([]*store.Comment, 0)
	commentDivs.Each(func(i int, commentDiv *goquery.Selection) {
		numRepliesStr := commentDiv.AttrOr("data-replies", "0")
		numReplies, err := strconv.Atoi(numRepliesStr)
		if err != nil {
			logEntry.WithError(err).Error("Couldn't parse number of replies, assuming 0")
			numReplies = 0
		}
		// fmt.Println(selection.First().Text())
		if numReplies == 0 {
			// If comment has no replies, just get the comment itself
			comments = append(comments, newCommentFromDiv(ctx, commentDiv, article.ID, userIDToNameMap))
		} else { // If the comment has replies, check for a "Load More" button
			loadMore := commentDiv.Find("button.comments-more")

			// If the button doesn't exist, just get the comments
			if loadMore.Length() == 0 {
				// WARNING
				// The logic for the rest of this IF is different than the original scraper
				// Something has changed or I'm bad at converting Clojure to Go

				// get the parent comment
				comments = append(comments, newCommentFromDiv(ctx, commentDiv, article.ID, userIDToNameMap))

				// get the replies
				comments = append(comments, getReplies(ctx, article, strconv.Itoa(getCommentID(ctx, commentDiv)), userIDToNameMap)...)

			} else { // If the button exists, get all replies
				// TODO - the reply endpoint is different, I'm not sure what happens if there are more than 20 replies
				// Log added to deal with this
				if numReplies > 20 {
					logEntry.Warn("HUGE REPLY CHAIN FOUND!")
				} else {
					comments = append(comments, getReplies(ctx, article, loadMore.AttrOr("data-parent", ""), userIDToNameMap)...)
				}
			}
		}
	})

	return comments
}

// TODO - lastId is currently unused because I've yet to see a reply chain > 20 comments
// We can nly surmise on the usage currently
func getReplies(ctx context.Context, article *store.Article, parentID string, userIDToNameMap map[int]string) []*store.Comment {
	logEntry := sdk.Logger(ctx).WithField("article_id", article.ID)
	baseUrl, _ := getBaseUrl(article.Url) // not worried about error because we would have errored on this in the parent function
	commentsUrl := fmt.Sprintf("%s/comments/get?ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest&lastId=%22%22&ParentId=%s", baseUrl, article.ID, parentID)
	res, err := http.Get(commentsUrl)
	if err != nil {
		logEntry.WithError(err).Fatal("Failed to load comments")
		return nil
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			logEntry.WithError(err).Error("Error closing HTTP")
		}
	}()

	if res.StatusCode != 200 {
		logEntry.WithFields(logrus.Fields{
			"status_code": res.StatusCode,
			"status":      res.StatusCode,
		}).Fatal("status code error")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logEntry.WithError(err).Error("Error loading HTTP response body")
		return nil
	}

	docComments := doc.Find("div.comment")
	comments := make([]*store.Comment, 0)
	docComments.Each(func(i int, reply *goquery.Selection) {
		comments = append(comments, newCommentFromDiv(ctx, reply, article.ID, userIDToNameMap))
	})
	return comments
}

func getContentHelper(s *goquery.Selection) string {
	return strings.TrimSpace(s.First().Text())
}

func getCommentID(ctx context.Context, s *goquery.Selection) int {
	logEntry := sdk.Logger(ctx)
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
		sdk.Logger(ctx).WithError(err).Error("Couldn't parse user ID, using 0")
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
		sdk.Logger(ctx).WithError(err).Error("Couldn't parse time, using current time")
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
		sdk.Logger(ctx).WithError(err).Error("Unable to get likes")
		return 0
	}

	return int32(likes)
}

func getDislikes(ctx context.Context, s *goquery.Selection) int32 {
	dislikeString := getContentHelper(s.Find("[value=Downvote]"))
	dislikes, err := strconv.Atoi(dislikeString)
	if err != nil {
		sdk.Logger(ctx).WithError(err).Error("Unable to get dislikes")
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

// NOTE - this relies on the fact that on SooToday's comments they do not consider replies as comments
// The number of comments is == to the top level comments
// It also relies on the fact that only 20 comments can be displayed per article and this includes replies
func getNumberOfCommentApiCalls(ctx context.Context, doc *goquery.Document) int {
	topLevelCommentCountStr := doc.Find("div#comments").First().AttrOr("data-count", "0")
	topLevelCommentCount, err := strconv.Atoi(topLevelCommentCountStr)
	if err != nil {
		sdk.Logger(ctx).WithError(err).Error("Error converting data-count to int")
		return 0
	}

	return int(math.Ceil(float64(topLevelCommentCount) / 20))
}

func getArticleId(ctx context.Context, url string) int {
	articleIdStr := url[strings.LastIndex(url, "-")+1:]
	articleId, err := strconv.Atoi(articleIdStr)
	if err != nil {
		sdk.Logger(ctx).WithError(err).Error("Error converting article ID to int, using 0")
		return 0
	}
	return articleId
}

func ScrapeArticles(ctx context.Context, siteUrl string) []*store.Article {
	logEntry := sdk.Logger(ctx).WithField("site_url", siteUrl)
	res, err := http.Get(siteUrl)
	if err != nil {
		logEntry.WithError(err).Fatal("Failed to load articles")
		return nil
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			logEntry.WithError(err).Error("Error closing HTTP")
		}
	}()

	if res.StatusCode != 200 {
		logEntry.WithFields(logrus.Fields{
			"status_code": res.StatusCode,
			"status":      res.StatusCode,
		}).Fatal("status code error")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logEntry.WithError(err).Error("Error loading article")
	}

	var articles []*store.Article
	doc.Find("a.section-item").Each(func(i int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists {
			if isArticleUrl(href) {
				title := sel.Find("div.section-title").First().Text()
				// sometimes theres numbers at the end of titles - clean em up
				title = strings.TrimSpace(title)
				title = strings.TrimRightFunc(title, func(r rune) bool {
					return unicode.IsNumber(r) || unicode.IsSpace(r)
				})
				articles = append(articles, &store.Article{
					ID:             getArticleId(ctx, href),
					Title:          title,
					Url:            siteUrl + href,
					DiscoveryTime:  time.Now(),
					LastScrapeTime: time.Now(),
				})
			}
		}
	})

	articleIDs := make([]int, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	logEntry.WithField("articles", articleIDs).Info("Articles found")
	return articles
}
