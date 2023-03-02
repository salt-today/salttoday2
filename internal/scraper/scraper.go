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

type Scraper struct {
	logger  logrus.Logger
	siteUrl string
}

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

func NewScraper(siteUrl string) *Scraper {
	return &Scraper{
		siteUrl: siteUrl,
	}
}

func (s *Scraper) Scrape(ctx context.Context, siteUrl string) {
	logEntry := sdk.Logger(ctx).WithField("site", siteUrl)

	articles := getArticles(ctx, siteUrl)
	commentCount := 0
	for _, article := range articles {
		comments, err := getCommentsFromArticle(ctx, siteUrl, article)
		if err != nil {
			logEntry.WithError(err).Error("failed to get comments")
			continue
		}

		for _, comment := range comments {
			commentCount++
			logEntry.WithField("comment", comment).Info("retrieved comment")
		}
	}
	logEntry.WithFields(logrus.Fields{
		"articles": len(articles),
		"comments": commentCount,
	}).Info("Completed scraping site")
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
func getCommentsFromArticle(ctx context.Context, baseUrl string, article *store.Article) ([]*store.Comment, error) {
	articleId := getArticleId(article.Url)
	logEntry := sdk.Logger(ctx).WithField("articleId", articleId)
	commentsUrl := fmt.Sprintf("%s/comments/load?Type=Comment&ContentId=%s&TagId=2346&TagType=Content&Sort=Oldest", baseUrl, articleId)

	res, err := http.Get(commentsUrl)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to download comments")
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
			"status":      res.StatusCode,
		}).Fatal("status code error")
		return nil, nil
	}

	initialCommentDoc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logEntry.WithError(err).Error("Error loading HTTP response body")
		return nil, err
	}

	numberOfCommentApiCalls := getNumberOfCommentApiCalls(ctx, initialCommentDoc)

	comments := searchArticleForComments(ctx, numberOfCommentApiCalls, initialCommentDoc, baseUrl, articleId)
	return comments, nil
}

func searchArticleForComments(ctx context.Context, numberOfCommentApiCalls int, initialCommentDoc *goquery.Document, baseUrl string, articleId string) []*store.Comment {
	if numberOfCommentApiCalls <= 0 {
		return []*store.Comment{}
	}

	// Grab all top level comments
	commentsDoc := selectTopLevelComments(initialCommentDoc)
	comments := getComments(ctx, commentsDoc, baseUrl, articleId)

	return comments
}

func getComments(ctx context.Context, selection *goquery.Selection, baseUrl, articleId string) []*store.Comment {
	logEntry := sdk.Logger(ctx).WithField("article_id", articleId)
	comments := make([]*store.Comment, 0)
	selection.Each(func(i int, s *goquery.Selection) {
		numRepliesStr := s.AttrOr("data-replies", "0")
		numReplies, err := strconv.Atoi(numRepliesStr)
		if err != nil {
			// fmt.Println("Couldn't parse number of replies, assuming 0", err)
			numReplies = 0
		}
		// fmt.Println(selection.First().Text())
		if numReplies == 0 {
			// If comment has no replies, just get the comment itself
			comments = append(comments, &store.Comment{
				User:     getUsername(ctx, s),
				Time:     getTimestamp(ctx, s),
				Text:     getCommentText(ctx, s),
				Likes:    getLikes(ctx, s),
				Dislikes: getDislikes(ctx, s),
			})
		} else { // If the comment has replies, check for a "Load More" button
			loadMore := s.Find("button.comments-more")

			// If the button doesn't exist, just get the comments
			if loadMore.Length() == 0 {
				// WARNING
				// The logic for the rest of this IF is different than the original scraper
				// Something has changed or I'm bad at converting Clojure to Go

				// get the parent comment
				comments = append(comments, &store.Comment{
					User:     getUsername(ctx, s),
					Time:     getTimestamp(ctx, s),
					Text:     getCommentText(ctx, s),
					Likes:    getLikes(ctx, s),
					Dislikes: getDislikes(ctx, s),
				})

				// get the replies
				comments = append(comments, getReplies(ctx, baseUrl, articleId, "-1", s.AttrOr("data-id", ""))...)

			} else { // If the button exists, get all replies
				// TODO - the reply endpoint is different, I'm not sure what happens if there are more than 20 replies
				// Log added to deal with this
				if numReplies > 20 {
					logEntry.Warn("HUGE REPLY CHAIN FOUND!")
				} else {
					comments = append(comments, getReplies(ctx, baseUrl, articleId, "-1", loadMore.AttrOr("data-parent", ""))...)
				}
			}
		}
	})

	return comments
}

// TODO - lastId is currently unused because I've yet to see a reply chain > 20 comments
// We can nly surmise on the usage currently
func getReplies(ctx context.Context, baseUrl, articleId, _, parentId string) []*store.Comment {
	logEntry := sdk.Logger(ctx).WithField("article_id", articleId)
	commentsUrl := fmt.Sprintf("%s/comments/get?ContentId=%s&TagId=2346&TagType=Content&Sort=Oldest&lastId=%22%22&ParentId=%s", baseUrl, articleId, parentId)
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
		comments = append(comments, &store.Comment{
			User:     getUsername(ctx, reply),
			Time:     getTimestamp(ctx, reply),
			Text:     getCommentText(ctx, reply),
			Likes:    getLikes(ctx, reply),
			Dislikes: getDislikes(ctx, reply),
		})
	})
	return comments
}

func getContentHelper(s *goquery.Selection) string {
	return strings.TrimSpace(s.First().Text())
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

func getLikes(ctx context.Context, s *goquery.Selection) int {
	likeString := getContentHelper(s.Find("[value=Upvote]"))
	likes, err := strconv.Atoi(likeString)
	if err != nil {
		sdk.Logger(ctx).WithError(err).Error("Unable to get likes")
		return 0
	}

	return likes
}

func getDislikes(ctx context.Context, s *goquery.Selection) int {
	dislikeString := getContentHelper(s.Find("[value=Downvote]"))
	dislikes, err := strconv.Atoi(dislikeString)
	if err != nil {
		sdk.Logger(ctx).WithError(err).Error("Unable to get dislikes")
		return 0
	}

	return dislikes
}

// SooToday has two types of return values from their API
// 1. contains a div.comments element - returned from a page load
// 2. just contains a body tag with a collection of comments
func selectTopLevelComments(commentsDoc *goquery.Document) *goquery.Selection {
	// This isn't completely necessary anymore because I found a selector that works for both
	// However, separating this might still be a good idea incase sootoday changes and allows for deeply nested replies.
	// This only works but only top level comments have the data-replies attr

	// fmt.Println(goquery.OuterHtml(commentsDoc.Selection))
	s := commentsDoc.Find("div#comments")
	if s.Length() == 0 { // is this right? we were checking if empty in clj
		return s.Find("div[data-replies]") // HERE DAN HERE - I'm pretty sure I did this wrong
	}
	return s.Find("div#comments > div.comment")
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

func getArticleId(url string) string {
	return url[strings.LastIndex(url, "-")+1:]
}

func getArticles(ctx context.Context, siteUrl string) []*store.Article {
	logEntry := sdk.Logger(ctx)
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
	doc.Find("a.section-item").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if hasArticlePrefix(href) {
				title := s.Find("div.section-title").First().Text()
				title = strings.TrimSpace(title)
				articles = append(articles, &store.Article{
					Title: title,
					Url:   siteUrl + href,
				})
			}
		}
	})
	logEntry.WithField("articles", articles).Info("Articles found")
	return articles
}
