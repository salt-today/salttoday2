package scraper

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb"
)

// TestViafouraLive hits the real Viafoura API. It is skipped unless VIAFOURA_LIVE
// is set, so it never runs in normal `go test`. Run with:
//
//	VIAFOURA_LIVE=1 go test ./internal/scraper -run TestViafouraLive -v
func TestViafouraLive(t *testing.T) {
	if os.Getenv("VIAFOURA_LIVE") == "" {
		t.Skip("set VIAFOURA_LIVE=1 to run the live Viafoura test")
	}

	vf := newViafouraScraper(&http.Client{Timeout: 30 * time.Second}, DefaultConfig())

	// The dark-horse coffee article: a known-commented lfpress article.
	article := &store.Article{
		ID:       "0829a1b9-e996-49d6-b670-406436855e68", // vf:container_id
		SiteName: "LondonFreePress",
		Url:      "https://lfpress.com/business/local-business/toronto-based-coffee-chain-dark-horse-opening-second-london-spot",
	}

	comments, users, err := vf.ScrapeComments(context.Background(), article)
	if err != nil {
		t.Fatalf("ScrapeComments: %v", err)
	}

	if len(comments) == 0 {
		t.Fatalf("expected comments, got none")
	}

	nameByID := make(map[string]string)
	for _, u := range users {
		nameByID[u.ID] = u.UserName
	}

	t.Logf("scraped %d comments, %d users", len(comments), len(users))
	for _, c := range comments {
		t.Logf("  [%s] %s (👍%d 👎%d) by %s: %.60s", c.ID[:8], c.Time.Format("2006-01-02"), c.Likes, c.Dislikes, nameByID[c.User.ID], c.Text)
	}
}

// TestStoreRoundTripLive scrapes a real lfpress (Viafoura) article and a real
// SooToday (Village) article, stores both, and reads them back — proving comments
// from both site types persist to the DB. Requires a disposable MySQL:
//
//	MYSQL_URL="root:salt@tcp(localhost:3307)/salt" STORE_LIVE=1 \
//	  go test ./internal/scraper -run TestStoreRoundTripLive -v -timeout 300s
func TestStoreRoundTripLive(t *testing.T) {
	if os.Getenv("STORE_LIVE") == "" {
		t.Skip("set STORE_LIVE=1 (and MYSQL_URL to a disposable DB) to run")
	}
	ctx := context.Background()

	storage, err := rdb.New(ctx)
	if err != nil {
		t.Fatalf("rdb.New (migrations): %v", err)
	}

	s, err := NewScraper(ctx, nil)
	if err != nil {
		t.Fatalf("NewScraper: %v", err)
	}
	defer s.Close()

	storeAll := func(label string, article *store.Article, comments []*store.Comment, users []*store.User) {
		if err := storage.AddArticles(ctx, article); err != nil {
			t.Fatalf("%s AddArticles: %v", label, err)
		}
		if err := storage.AddUsers(ctx, users...); err != nil {
			t.Fatalf("%s AddUsers: %v", label, err)
		}
		if err := storage.AddComments(ctx, comments); err != nil {
			t.Fatalf("%s AddComments: %v", label, err)
		}
		t.Logf("%s: stored article=%s with %d comments, %d users", label, article.ID, len(comments), len(users))
	}

	readBack := func(label, articleID string) int {
		got, err := storage.GetComments(ctx, &store.CommentQueryOptions{
			ArticleID: &articleID,
			PageOpts:  &store.PageQueryOptions{Order: store.OrderByBoth, Limit: ptrUint(100)},
		})
		if err != nil {
			t.Fatalf("%s GetComments: %v", label, err)
		}
		c := got[0]
		t.Logf("%s: read back %d comments; sample id=%s user=%s(%s) 👍%d 👎%d site=%s",
			label, len(got), c.ID, c.User.ID, c.User.UserName, c.Likes, c.Dislikes, c.Article.SiteName)
		return len(got)
	}

	// --- Viafoura (lfpress) ---
	vfArticle := &store.Article{
		ID:       "0829a1b9-e996-49d6-b670-406436855e68",
		SiteName: "LondonFreePress",
		Url:      "https://lfpress.com/business/local-business/toronto-based-coffee-chain-dark-horse-opening-second-london-spot",
		Title:    "Dark Horse coffee opening second London spot",
	}
	vfComments, vfUsers, err := s.strategies[internal.SiteTypeViafoura].ScrapeComments(ctx, vfArticle)
	if err != nil {
		t.Fatalf("viafoura scrape: %v", err)
	}
	storeAll("lfpress", vfArticle, vfComments, vfUsers)

	// --- Village (SooToday): pick a discovered article that has comments ---
	village := s.strategies[internal.SiteTypeVillage]
	vArticles, err := village.DiscoverArticles(ctx, internal.SitesMap["SooToday"])
	if err != nil {
		t.Fatalf("village discover: %v", err)
	}
	var vArticle *store.Article
	var vComments []*store.Comment
	var vUsers []*store.User
	for i, a := range vArticles {
		if i >= 15 {
			break
		}
		c, u, err := village.ScrapeComments(ctx, a)
		if err == nil && len(c) > 0 {
			vArticle, vComments, vUsers = a, c, u
			break
		}
	}
	if vArticle == nil {
		t.Fatalf("no SooToday article with comments found to store")
	}
	storeAll("sootoday", vArticle, vComments, vUsers)

	// --- read both back from the DB ---
	gotVF := readBack("lfpress", vfArticle.ID)
	gotV := readBack("sootoday", vArticle.ID)

	if gotVF == 0 || gotV == 0 {
		t.Fatalf("expected both site types persisted; lfpress=%d sootoday=%d", gotVF, gotV)
	}
}

func ptrUint(u uint) *uint { return &u }

// TestVillageLive verifies the refactored Village Media path still scrapes
// articles + comments end-to-end (uses a real browser). Run with:
//
//	VILLAGE_LIVE=1 go test ./internal/scraper -run TestVillageLive -v -timeout 300s
func TestVillageLive(t *testing.T) {
	if os.Getenv("VILLAGE_LIVE") == "" {
		t.Skip("set VILLAGE_LIVE=1 to run the live Village Media test")
	}

	s, err := NewScraper(context.Background(), nil)
	if err != nil {
		t.Fatalf("NewScraper: %v", err)
	}
	defer s.Close()

	site := internal.SitesMap["SooToday"]
	village := s.strategies[internal.SiteTypeVillage]

	articles, err := village.DiscoverArticles(context.Background(), site)
	if err != nil {
		t.Fatalf("DiscoverArticles: %v", err)
	}
	if len(articles) == 0 {
		t.Fatalf("expected Village articles, got none")
	}
	t.Logf("discovered %d %s articles (sample id=%s url=%s)", len(articles), site.Name, articles[0].ID, articles[0].Url)

	totalComments, withComments := 0, 0
	limit := 15
	for i, a := range articles {
		if i >= limit {
			break
		}
		comments, users, err := village.ScrapeComments(context.Background(), a)
		if err != nil {
			t.Logf("  ! %s: %v", a.Url, err)
			continue
		}
		if len(comments) > 0 {
			withComments++
			totalComments += len(comments)
			t.Logf("  %3d comments  %s", len(comments), a.Title)
			c := comments[0]
			t.Logf("        e.g. id=%s user=%s 👍%d 👎%d: %.50s", c.ID, c.User.ID, c.Likes, c.Dislikes, c.Text)
			_ = users
		}
	}
	t.Logf("=== Village: %d comments across %d/%d articles checked ===", totalComments, withComments, limit)
	if totalComments == 0 {
		t.Errorf("expected at least some Village comments across %d articles", limit)
	}
}

func TestViafouraSiteWideLive(t *testing.T) {
	if os.Getenv("VIAFOURA_LIVE") == "" {
		t.Skip("set VIAFOURA_LIVE=1 to run the live Viafoura test")
	}

	vf := newViafouraScraper(&http.Client{Timeout: 30 * time.Second}, DefaultConfig())
	site := internal.SitesMap["LondonFreePress"]

	articles, err := vf.DiscoverArticles(context.Background(), site)
	if err != nil {
		t.Fatalf("DiscoverArticles: %v", err)
	}
	t.Logf("discovered %d articles; scraping comments for each...", len(articles))

	totalComments := 0
	articlesWithComments := 0
	allUsers := make(map[string]string)
	for _, a := range articles {
		comments, users, err := vf.ScrapeComments(context.Background(), a)
		if err != nil {
			t.Logf("  ! %s: %v", a.Url, err)
			continue
		}
		if len(comments) > 0 {
			articlesWithComments++
			totalComments += len(comments)
			for _, u := range users {
				allUsers[u.ID] = u.UserName
			}
			t.Logf("  %3d comments  %s", len(comments), a.Title)
		}
	}

	t.Logf("=== TOTAL: %d comments across %d/%d articles, %d distinct commenters ===",
		totalComments, articlesWithComments, len(articles), len(allUsers))
}

func TestViafouraDiscoverLive(t *testing.T) {
	if os.Getenv("VIAFOURA_LIVE") == "" {
		t.Skip("set VIAFOURA_LIVE=1 to run the live Viafoura test")
	}

	vf := newViafouraScraper(&http.Client{Timeout: 30 * time.Second}, DefaultConfig())
	site := internal.SitesMap["LondonFreePress"]

	articles, err := vf.DiscoverArticles(context.Background(), site)
	if err != nil {
		t.Fatalf("DiscoverArticles: %v", err)
	}
	if len(articles) == 0 {
		t.Fatalf("expected articles from sitemap, got none")
	}

	t.Logf("discovered %d articles", len(articles))
	for i, a := range articles {
		if i >= 5 {
			break
		}
		t.Logf("  id=%s title=%q url=%s", a.ID, a.Title, a.Url)
		if a.ID == "" || a.Title == "" {
			t.Errorf("article missing id or title: %+v", a)
		}
		if a.SiteName != "LondonFreePress" {
			t.Errorf("unexpected site name %q", a.SiteName)
		}
	}
}
