package main

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

func handler(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("lambda", "scrapper/comments")
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	// Get articles from the last 7 days
	articles, err := storage.GetUnscrapedArticlesSince(ctx, time.Now().Add(-time.Hour*24*7))
	if err != nil {
		logEntry.WithError(err).Fatal("failed to get article ids")
	}

	var articleIDs = lo.FlatMap[*store.Article, int](articles, func(item *store.Article, _ int) []int {
		return []int{item.ID}
	})

	comments := scraper.ScrapeComments(ctx, articleIDs)
	for _, comment := range comments {
		logEntry.WithField("comment", comment.ID).Info("Found comment")
	}
	storage.AddComments(ctx, comments...)
}

func main() {
	// TODO remove once we have better local testing story
	// handler(context.Background())

	lambda.Start(handler)
}
