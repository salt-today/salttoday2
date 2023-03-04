package main

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

func handler(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("lambda", "scrapper/comments")
	store, err := store.NewStorage()
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Get articles from the last 7 days
	articleIDs, err := store.GetUnscrapedArticlesSince(time.Now().Add(-time.Hour * 24 * 7))
	if err != nil {
		logEntry.WithError(err).Fatal("failed to get article ids")
	}

	comments := scraper.ScrapeComments(ctx, articleIDs)
	for _, comment := range comments {
		logEntry.WithField("comment", comment.ID).Info("Found comment")
	}
	store.AddComments(comments...)
}

func main() {
	// TODO remove once we have better local testing story
	// handler(context.Background())

	lambda.Start(handler)
}
