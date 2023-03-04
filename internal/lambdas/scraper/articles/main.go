package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

type scrapeArticlesEvent struct {
	siteUrl string `json:"site_url"`
}

func handler(ctx context.Context, event scrapeArticlesEvent) {
	logEntry := sdk.Logger(ctx).WithField("lambda", "scrapper/articles")
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	articles := scraper.ScrapeArticles(ctx, event.siteUrl)
	for _, article := range articles {
		logEntry.WithField("article", article.ID).Info("Found article")
	}

	err = storage.AddArticles(ctx, articles...)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to add articles")
	}
}

func main() {
	// TODO remove once we have better local testing story
	// handler(context.Background(), scrapeArticlesEvent{siteUrl: "https://tbnewswatch.com"})

	lambda.Start(handler)
}
