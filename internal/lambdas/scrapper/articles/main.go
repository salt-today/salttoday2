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
	store, err := store.NewStorage()
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	articles := scraper.ScrapeArticles(ctx, event.siteUrl)
	store.AddArticles(articles...)
}

func main() {
	lambda.Start(handler)
}