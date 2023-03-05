package jobs

import (
	"context"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

func ScrapeAndStoreArticles(ctx context.Context) {
	logEntry := sdk.Logger(ctx).WithField("job", "scraper/articles")
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to create storage")
	}

	articles := make([]*store.Article, 0)
	for _, site := range internal.GetSites() {
		foundArticles := scraper.ScrapeArticles(ctx, site)
		for _, article := range foundArticles {
			logEntry.WithField("article", article.ID).Info("Found article")
		}
		articles = append(articles, foundArticles...)
	}
	err = storage.AddArticles(ctx, articles...)
	if err != nil {
		logEntry.WithError(err).Fatal("failed to add articles")
	}
}
