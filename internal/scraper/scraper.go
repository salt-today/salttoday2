package scraper

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

// verify these are still correct and if there's any missing
var potentialPrefixes = []string{
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

func ScrapeComments(ctx context.Context, articleIDs []int) []*store.Comment {
	logEntry := sdk.Logger(ctx)

	comments := make([]*store.Comment, 0)
	for _, articleID := range articleIDs {
		articleComments, err := getCommentsFromArticle(ctx, articleID)
		if err != nil {
			logEntry.WithError(err).Error("failed to get comments")
			continue
		}

		comments = append(comments, articleComments...)

		for _, comment := range comments {
			logEntry.WithField("comment", comment).Info("retrieved comment")
		}
	}
	logEntry.WithFields(logrus.Fields{
		"articles": len(articleIDs),
		"comments": len(comments),
	}).Info("Completed scraping site")

	return comments
}

func hasArticlePrefix(href string) bool {
	for _, prefix := range potentialPrefixes {
		if strings.HasPrefix(href, prefix) {
			return true
		}
	}
	return false
}

// This logic is ported from the old scraper so blame Tyler if it's bad
// The api for sootoday comments is awful so also thanks Tyler for figuring this out
func getCommentsFromArticle(ctx context.Context, articleID int) ([]*store.Comment, error) {
	logEntry := sdk.Logger(ctx).WithField("articleId", articleID)
	// Base url hardcoded to sootoday because it literally doesn't matter what the base url is
	commentsUrl := fmt.Sprintf("https://www.sootoday.com/comments/load?Type=Comment&ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest", articleID)

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

	numberOfCommentApiCalls := getNumberOfCommentApiCalls(ctx, initialCommentDoc)

	comments := searchArticleForComments(ctx, numberOfCommentApiCalls, initialCommentDoc, articleID)
	return comments, nil
}

func searchArticleForComments(ctx context.Context, numberOfCommentApiCalls int, initialCommentDoc *goquery.Document, articleID int) []*store.Comment {
	if numberOfCommentApiCalls <= 0 {
		return []*store.Comment{}
	}

	// Grab all top level comments
	commentDivs := selectTopLevelComments(initialCommentDoc)
	comments := getComments(ctx, commentDivs, articleID)

	return comments
}

func getComments(ctx context.Context, commentDivs *goquery.Selection, articleID int) []*store.Comment {
	logEntry := sdk.Logger(ctx).WithField("article_id", articleID)
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
			comments = append(comments, newCommentFromDiv(ctx, commentDiv))
		} else { // If the comment has replies, check for a "Load More" button
			loadMore := commentDiv.Find("button.comments-more")

			// If the button doesn't exist, just get the comments
			if loadMore.Length() == 0 {
				// WARNING
				// The logic for the rest of this IF is different than the original scraper
				// Something has changed or I'm bad at converting Clojure to Go

				// get the parent comment
				comments = append(comments, newCommentFromDiv(ctx, commentDiv))

				// get the replies
				comments = append(comments, getReplies(ctx, articleID, strconv.Itoa(getCommentID(ctx, commentDiv)))...)

			} else { // If the button exists, get all replies
				// TODO - the reply endpoint is different, I'm not sure what happens if there are more than 20 replies
				// Log added to deal with this
				if numReplies > 20 {
					logEntry.Warn("HUGE REPLY CHAIN FOUND!")
				} else {
					comments = append(comments, getReplies(ctx, articleID, loadMore.AttrOr("data-parent", ""))...)
				}
			}
		}
	})

	return comments
}

// TODO - lastId is currently unused because I've yet to see a reply chain > 20 comments
// We can nly surmise on the usage currently
func getReplies(ctx context.Context, articleID int, parentID string) []*store.Comment {
	logEntry := sdk.Logger(ctx).WithField("article_id", articleID)
	commentsUrl := fmt.Sprintf("https://www.sootoday.com/comments/get?ContentId=%d&TagId=2346&TagType=Content&Sort=Oldest&lastId=%22%22&ParentId=%s", articleID, parentID)
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
		comments = append(comments, newCommentFromDiv(ctx, reply))
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
	return commentTime // TODO
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

func newCommentFromDiv(ctx context.Context, div *goquery.Selection) *store.Comment {
	return &store.Comment{
		ID:       getCommentID(ctx, div),
		UserID:   getUserID(ctx, div),
		Name:     getUsername(ctx, div),
		Time:     getTimestamp(ctx, div),
		Text:     getCommentText(ctx, div),
		Likes:    getLikes(ctx, div),
		Dislikes: getDislikes(ctx, div),
	}
}

// SooToday has two types of return values from their API
// 1. contains a div.comments element - returned from a page load
// 2. just contains a body tag with a collection of comments
func selectTopLevelComments(commentsDoc *goquery.Document) *goquery.Selection {
	// This isn't completely necessary anymore because I found a selector that works for both
	// However, separating this might still be a good idea incase sootoday changes and allows for deeply nested replies.
	// This only works but only top level comments have the data-replies attr

	// fmt.Println(goquery.OuterHtml(commentsDoc.Selection))
	commentsDiv := commentsDoc.Find("div#comments")
	if commentsDiv.Length() == 0 { // is this right? we were checking if empty in clj
		return commentsDiv.Find("div[data-replies]") // HERE DAN HERE - I'm pretty sure I did this wrong
	}
	return commentsDiv.Find("div#comments > div.comment")
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
			if hasArticlePrefix(href) {
				title := sel.Find("div.section-title").First().Text()
				title = strings.TrimSpace(title)
				articles = append(articles, &store.Article{
					ID:    getArticleId(ctx, href),
					Title: title,
					Url:   siteUrl + href,
				})
			}
		}
	})
	logEntry.WithField("articles", articles).Info("Articles found")
	return articles
}
