package scraper

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/store"
)

const (
	viafouraLiveComments = "https://livecomments.viafoura.co/v4/livecomments"
	viafouraAPI          = "https://api.viafoura.co/v2"
	viafouraUserAgent    = "Mozilla/5.0 (compatible; salttoday-scraper)"
)

// viafouraScraper scrapes Postmedia properties (e.g. lfpress.com), whose comments
// are served by Viafoura. Everything is plain HTTP, no browser:
//   - articles are discovered from the site's sitemap
//   - each article's Viafoura container UUID comes from its page's vf:container_id meta
//   - comments come from the public Viafoura REST API
//
// See internal/memory notes / cmd/scraper/viafoura-uuid for how the constants were found.
type viafouraScraper struct {
	client *http.Client
	config *ScrapingConfig
}

func newViafouraScraper(client *http.Client, config *ScrapingConfig) *viafouraScraper {
	return &viafouraScraper{client: client, config: config}
}

// --- sitemap discovery ---

// maxRecentSitemaps bounds how many child sitemaps of a sitemap index are
// followed. Children are dated and listed newest first, and the comment scraper
// only needs recent articles.
const maxRecentSitemaps = 5

type sitemapIndex struct {
	Sitemaps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

type urlSet struct {
	URLs []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

func (vf *viafouraScraper) DiscoverArticles(ctx context.Context, site internal.SiteConfig) ([]*store.Article, error) {
	logEntry := logger.New(ctx).WithField("site", site.Name)

	urls, err := vf.collectArticleURLs(ctx, site.SitemapURL, 0)
	if err != nil {
		return nil, &ScrapingError{Op: "FetchSitemap", URL: site.SitemapURL, Err: err.Error()}
	}
	logEntry.WithField("urls", len(urls)).Info("Sitemap article URLs found")

	// Fetch each article page (bounded concurrency) to extract its title and
	// Viafoura container UUID. Already-known articles are de-duplicated downstream
	// by INSERT IGNORE.
	var (
		articles []*store.Article
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, vf.config.MaxArticleWorkers)

	for _, articleURL := range urls {
		wg.Add(1)
		go func(articleURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			containerID, title, err := vf.fetchArticleMeta(ctx, articleURL)
			if err != nil {
				logEntry.WithError(err).WithField("url", articleURL).Warn("Failed to read article meta, skipping")
				return
			}
			if containerID == "" {
				return
			}

			mu.Lock()
			articles = append(articles, &store.Article{
				ID:             containerID,
				Title:          title,
				SiteName:       site.Name,
				Url:            articleURL,
				DiscoveryTime:  time.Now(),
				LastScrapeTime: time.Now(),
			})
			mu.Unlock()
		}(articleURL)
	}
	wg.Wait()

	return articles, nil
}

// collectArticleURLs fetches a sitemap (handling both <sitemapindex> and <urlset>)
// and returns article page URLs. depth guards against runaway nested indexes.
func (vf *viafouraScraper) collectArticleURLs(ctx context.Context, sitemapURL string, depth int) ([]string, error) {
	if depth > 3 {
		return nil, nil
	}

	body, err := vf.get(ctx, sitemapURL)
	if err != nil {
		return nil, err
	}

	// A sitemap index points at child sitemaps.
	var idx sitemapIndex
	if err := xml.Unmarshal(body, &idx); err == nil && len(idx.Sitemaps) > 0 {
		// Indexes list child sitemaps newest first and reach back decades
		// (lfpress goes to 1999), so only follow the most recent ones.
		children := idx.Sitemaps
		if len(children) > maxRecentSitemaps {
			children = children[:maxRecentSitemaps]
		}
		var urls []string
		for _, sm := range children {
			child, err := vf.collectArticleURLs(ctx, sm.Loc, depth+1)
			if err != nil {
				logger.New(ctx).WithError(err).WithField("sitemap", sm.Loc).Warn("Failed child sitemap")
				continue
			}
			urls = append(urls, child...)
		}
		return urls, nil
	}

	var set urlSet
	if err := xml.Unmarshal(body, &set); err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(set.URLs))
	for _, u := range set.URLs {
		urls = append(urls, u.Loc)
	}
	return urls, nil
}

// fetchArticleMeta reads the article page and extracts its Viafoura container UUID
// (the comments key) and title.
func (vf *viafouraScraper) fetchArticleMeta(ctx context.Context, articleURL string) (containerID, title string, err error) {
	body, err := vf.get(ctx, articleURL)
	if err != nil {
		return "", "", err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", "", err
	}

	containerID = doc.Find(`meta[name="vf:container_id"]`).AttrOr("content", "")
	title = strings.TrimSpace(doc.Find("h1#articleTitle").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}
	return containerID, title, nil
}

// --- comment scraping ---

type vfContainerResolve struct {
	ContentContainerUUID string `json:"content_container_uuid"`
	TotalVisibleContent  int    `json:"total_visible_content"`
}

type vfComment struct {
	ContentUUID   string `json:"content_uuid"`
	ActorUUID     string `json:"actor_uuid"`
	Content       string `json:"content"`
	Time          int64  `json:"time"`
	TotalLikes    int32  `json:"total_likes"`
	TotalDislikes int32  `json:"total_dislikes"`
	State         string `json:"state"`
}

type vfCommentsResponse struct {
	MoreAvailable bool        `json:"more_available"`
	Contents      []vfComment `json:"contents"`
}

type vfUserResult struct {
	Result struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
}

// vfUserBatchResponse handles both response shapes the batch endpoint returns:
// a map keyed by request key, or (when keys look numeric) a JSON array.
type vfUserBatchResponse struct {
	Responses map[string]vfUserResult `json:"responses"`
}

func (vf *viafouraScraper) ScrapeComments(ctx context.Context, article *store.Article) ([]*store.Comment, []*store.User, error) {
	site, ok := internal.SitesMap[article.SiteName]
	if !ok || site.ViafouraSectionUUID == "" {
		return nil, nil, &ScrapingError{Op: "ResolveSection", URL: article.Url, Err: "missing Viafoura section config"}
	}
	section := site.ViafouraSectionUUID

	// article.ID is the publisher container id (vf:container_id). Resolve it to the
	// internal Viafoura comments container, and grab the comment count.
	resolved, err := vf.resolveContainer(ctx, section, article.ID)
	if err != nil {
		return nil, nil, &ScrapingError{Op: "ResolveContainer", URL: article.Url, Err: err.Error()}
	}
	if resolved.ContentContainerUUID == "" || resolved.TotalVisibleContent == 0 {
		return nil, nil, nil // nothing to scrape
	}

	comments, actorUUIDs, err := vf.fetchAllComments(ctx, section, resolved.ContentContainerUUID, article.ID)
	if err != nil {
		return nil, nil, err
	}

	users, err := vf.resolveUsers(ctx, site, actorUUIDs)
	if err != nil {
		// Usernames are best-effort; log and continue with whatever we have.
		logger.New(ctx).WithError(err).WithField("article_id", article.ID).Warn("Failed to resolve some usernames")
	}

	return comments, users, nil
}

func (vf *viafouraScraper) resolveContainer(ctx context.Context, section, containerID string) (*vfContainerResolve, error) {
	u := fmt.Sprintf("%s/%s/contentcontainer/id?container_id=%s", viafouraLiveComments, section, url.QueryEscape(containerID))
	body, err := vf.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var out vfContainerResolve
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// fetchAllComments pages the comments endpoint until exhausted. Replies are
// returned flat in `contents`, so all are collected into one slice. Returns the
// comments plus the set of distinct actor UUIDs (for username resolution).
func (vf *viafouraScraper) fetchAllComments(ctx context.Context, section, containerUUID, articleID string) ([]*store.Comment, map[string]struct{}, error) {
	var comments []*store.Comment
	seen := make(map[string]struct{})
	actorUUIDs := make(map[string]struct{})
	startingFrom := ""

	for page := 0; page < vf.config.MaxCommentPages; page++ {
		u := fmt.Sprintf("%s/%s/%s/comments?limit=100&reply_limit=100&sorted_by=oldest", viafouraLiveComments, section, containerUUID)
		if startingFrom != "" {
			u += "&starting_from=" + url.QueryEscape(startingFrom)
		}

		body, err := vf.get(ctx, u)
		if err != nil {
			return nil, nil, err
		}

		var resp vfCommentsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, nil, err
		}

		newInPage := 0
		for _, c := range resp.Contents {
			if _, ok := seen[c.ContentUUID]; ok {
				continue
			}
			seen[c.ContentUUID] = struct{}{}
			newInPage++
			startingFrom = c.ContentUUID

			actorUUIDs[c.ActorUUID] = struct{}{}
			comments = append(comments, &store.Comment{
				ID:       c.ContentUUID,
				Article:  store.Article{ID: articleID},
				User:     store.User{ID: viafouraUserID(c.ActorUUID)},
				Time:     time.UnixMilli(c.Time),
				Text:     c.Content,
				Likes:    c.TotalLikes,
				Dislikes: c.TotalDislikes,
				Deleted:  c.State != "visible",
			})
		}

		if !resp.MoreAvailable || newInPage == 0 {
			break
		}
	}

	return comments, actorUUIDs, nil
}

// resolveUsers batch-resolves Viafoura actor UUIDs to display names.
func (vf *viafouraScraper) resolveUsers(ctx context.Context, site internal.SiteConfig, actorUUIDs map[string]struct{}) ([]*store.User, error) {
	if len(actorUUIDs) == 0 {
		return nil, nil
	}

	domain, err := hostFromURL(site.BaseURL)
	if err != nil {
		return nil, err
	}
	sectionID := viafouraNumericID(site.ViafouraSectionUUID)

	// Build the batched request: { "requests": { "<key>": {"route":"/users/<id>","verb":"get"} }, "site": "<domain>" }.
	// Keys must be non-numeric, otherwise the API returns `responses` as a JSON array
	// instead of a keyed object. Each result carries its own id + name, so we read
	// those directly rather than mapping keys back to actors.
	type reqEntry struct {
		Route string `json:"route"`
		Verb  string `json:"verb"`
	}
	payload := struct {
		Requests map[string]reqEntry `json:"requests"`
		Site     string              `json:"site"`
	}{
		Requests: make(map[string]reqEntry),
		Site:     domain,
	}
	i := 0
	for actor := range actorUUIDs {
		payload.Requests["k"+strconv.Itoa(i)] = reqEntry{Route: "/users/" + viafouraUserID(actor), Verb: "get"}
		i++
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/?section=%d&session=false&json=%s", viafouraAPI, sectionID, url.QueryEscape(string(jsonBytes)))
	body, err := vf.get(ctx, u)
	if err != nil {
		return nil, err
	}

	var resp vfUserBatchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	users := make([]*store.User, 0, len(resp.Responses))
	for _, r := range resp.Responses {
		if r.Result.ID == 0 {
			continue
		}
		name := r.Result.Name
		if name == "" {
			name = "Anonymous"
		}
		users = append(users, &store.User{ID: strconv.FormatInt(r.Result.ID, 10), UserName: name})
	}
	return users, nil
}

// --- helpers ---

func (vf *viafouraScraper) get(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", viafouraUserAgent)

	resp, err := vf.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: status %d", u, resp.StatusCode)
	}
	return body, nil
}

// viafouraUserID extracts the numeric user id encoded in an actor UUID
// (00000000-0000-4000-8000-<12 hex>) and returns it as a decimal string.
func viafouraUserID(actorUUID string) string {
	return strconv.FormatInt(viafouraNumericID(actorUUID), 10)
}

// viafouraNumericID decodes the trailing 12 hex digits of a Viafoura-encoded UUID
// (its last segment) into the underlying numeric id.
func viafouraNumericID(encodedUUID string) int64 {
	idx := strings.LastIndex(encodedUUID, "-")
	if idx < 0 {
		return 0
	}
	n, err := strconv.ParseInt(encodedUUID[idx+1:], 16, 64)
	if err != nil {
		return 0
	}
	return n
}

func hostFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}
