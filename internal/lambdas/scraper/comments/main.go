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
	storer, err := store.NewSQLStorage(context.TODO())
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Get articles from the last 7 days
	articles, err := storer.GetUnscrapedArticlesSince(time.Now().Add(-time.Hour * 24 * 7))
	if err != nil {
		logEntry.WithError(err).Fatal("failed to get article ids")
		return
	}

	comments := scraper.ScrapeComments(ctx, articles...)
	for _, comment := range comments {
		logEntry.WithField("comment", comment.ID).Info("Found comment")
	}

	if err := storer.AddComments(comments...); err != nil {
		logEntry.WithError(err).Fatal("failed to get article ids")
	}
}

func main() {
	// TODO remove once we have better local testing story
	// handler(context.Background())

	lambda.Start(handler)
}
