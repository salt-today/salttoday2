package jobs

import (
	"context"
	"time"

	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

func ScrapeAndStoreComments(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("job", "scraper/comments")
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Get articles from the last 7 days
	articles, err := storage.GetRecentlyDiscoveredArticles(ctx, time.Now().Add(-time.Hour*24*7))
	if err != nil {
		logEntry.WithError(err).Fatal("failed to get article ids")
		return
	}

	comments, users := scraper.ScrapeCommentsFromArticles(ctx, articles)
	for _, comment := range comments {
		logEntry.WithField("comment", comment.ID).Info("Found comment")
	}
	for _, user := range users {
		logEntry.WithField("user", user.ID).Info("Found user")
	}

	if err := storage.AddComments(ctx, comments...); err != nil {
		logEntry.WithError(err).Error("failed to add comments ids")
	}

	if err := storage.AddUsers(ctx, users...); err != nil {
		logEntry.WithError(err).Error("failed to add user ids")
	}
}
