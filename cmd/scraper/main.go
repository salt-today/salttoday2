package main

import (
	"context"

	"github.com/salt-today/salttoday2/internal/scraper"
)

func main() {
	scraper.ScrapeAndStoreArticles(context.Background())

	scraper.ScrapeAndStoreComments(context.Background())
}
