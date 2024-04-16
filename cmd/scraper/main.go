package main

import (
	"context"

	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
)

func main() {
	ctx := context.Background()

	scraper.ScrapeAndStoreArticles(ctx)
	scraper.ScrapeAndStoreComments(ctx)

	sdk.Logger(ctx).Info("Scraping complete")
}
