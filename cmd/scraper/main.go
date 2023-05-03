package main

import (
	"context"
	"time"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

func main() {
	ScrapeAndStoreArticles(context.Background())
	
	ScrapeAndStoreComments(context.Background())
	
}

func ScrapeAndStoreArticles(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("job", "scraper/articles")
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	articles := make([]*store.Article, 0)
	for _, site := range internal.GetSites() {
		foundArticles := scraper.ScrapeArticles(ctx, site)
		articles = append(articles, foundArticles...)
	}
	err = storage.AddArticles(ctx, articles...)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to add articles")
	}
}

func ScrapeAndStoreComments(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("job", "scraper/comments")
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Get articles from the last 7 days
	articles, err := storage.GetUnscrapedArticlesSince(ctx, time.Now().Add(-time.Hour*24*7))
	if err != nil {
		logEntry.WithError(err).Error("failed to get article ids")
		return
	}

	comments, users := scraper.ScrapeCommentsFromArticles(ctx, articles)
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

	if err := storage.AddComments(ctx, comments...); err != nil {
		logEntry.WithError(err).Error("failed to add comments ids")
	}

	if err := storage.AddUsers(ctx, users...); err != nil {
		logEntry.WithError(err).Error("failed to add user ids")
	}
}
