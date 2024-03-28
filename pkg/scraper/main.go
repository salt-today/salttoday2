package scraper

import (
	"context"

	scraper "github.com/salt-today/salttoday2/pkg/scraper/internal"
)

func Main() {
	scraper.ScrapeAndStoreArticles(context.Background())

	scraper.ScrapeAndStoreComments(context.Background())

}
