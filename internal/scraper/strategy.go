package scraper

import (
	"context"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/store"
)

// SiteScraper encapsulates the site-specific scraping logic. Implementations:
//   - villageScraper (Village Media): village.go
//   - viafouraScraper (Postmedia/Viafoura): viafoura.go
type SiteScraper interface {
	// DiscoverArticles returns the articles currently advertised by the site, each
	// with SiteName set to site.Name.
	DiscoverArticles(ctx context.Context, site internal.SiteConfig) ([]*store.Article, error)
	// ScrapeComments returns all comments and their authors for a single article.
	ScrapeComments(ctx context.Context, article *store.Article) ([]*store.Comment, []*store.User, error)
}
