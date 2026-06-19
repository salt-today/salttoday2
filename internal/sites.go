package internal

import "sort"

func init() {
	SitesMapKeys = make([]string, 0, len(SitesMap))
	for key := range SitesMap {
		SitesMapKeys = append(SitesMapKeys, key)
	}
	sort.Strings(SitesMapKeys)
}

var SitesMapKeys []string

// Used for articles across multiple sites
const AllSitesName = "all"

// SiteType selects which scraping strategy a site uses.
type SiteType string

const (
	// SiteTypeVillage covers Village Media properties (homepage article listing +
	// the /comments/get AJAX endpoint).
	SiteTypeVillage SiteType = "village"
	// SiteTypeViafoura covers Postmedia properties (sitemap discovery + the public
	// Viafoura comments REST API). See internal/scraper/viafoura.go.
	SiteTypeViafoura SiteType = "viafoura"
)

// SiteConfig describes how to scrape a single site.
type SiteConfig struct {
	Name    string
	Type    SiteType
	BaseURL string

	// Viafoura-only configuration.
	SitemapURL string
	// ViafouraSectionUUID is a per-site constant. It is not present in page HTML;
	// it was captured once via cmd/scraper/viafoura-uuid and is hardcoded here.
	ViafouraSectionUUID string
}

func villageSite(name, baseURL string) SiteConfig {
	return SiteConfig{Name: name, Type: SiteTypeVillage, BaseURL: baseURL}
}

// TODO uncomment sites when we're ready, no need to hammer all of them yet
// List taken from here: villagemedia.ca/sites/
var SitesMap = map[string]SiteConfig{
	"TBNewsWatch": villageSite("TBNewsWatch", "https://www.tbnewswatch.com"),
	// "BarrieToday":   villageSite("BarrieToday", "https://www.barrietoday.com"),
	"BayToday": villageSite("BayToday", "https://www.baytoday.ca"),
	// "BradfordToday": villageSite("BradfordToday", "https://www.bradfordtoday.ca"),
	"GuelphToday": villageSite("GuelphToday", "https://www.guelphtoday.com"),
	// "innisfilToday": villageSite("innisfilToday", "https://www.innisfiltoday.ca"),
	// "MidlandToday": villageSite("MidlandToday", "https://midlandtoday.ca"),
	// "NewMarketToday": villageSite("NewMarketToday", "https://www.newmarkettoday.ca"),
	"OrilliaMatters": villageSite("OrilliaMatters", "https://www.orilliamatters.com"),
	"SooToday":       villageSite("SooToday", "https://www.sootoday.com"),
	// "StratfordToday": villageSite("StratfordToday", "https://www.stratfordtoday.ca"),
	// "TimminsToday": villageSite("TimminsToday", "https://www.timminstoday.com"),
	"PrinceGeorgeCitizen": villageSite("PrinceGeorgeCitizen", "https://www.princegeorgecitizen.com"),
	// "CoastReporter": villageSite("CoastReporter", "https://www.coastreporter.net"),
	// "TheReminder": villageSite("TheReminder", "https://www.thereminder.ca"),
	// "KamloopsThisWeek": villageSite("KamloopsThisWeek", "https://www.kamloopsthisweek.com"),
	// "MooseJawToday": villageSite("MooseJawToday", "https://www.moosejawtoday.com"),
	// "SquamishChief": villageSite("SquamishChief", "https://www.squamishchief.com"),
	"OkotoksToday": villageSite("OkotoksToday", "https://www.okotokstoday.ca"),

	// Postmedia / Viafoura
	"LondonFreePress": {
		Name:                "LondonFreePress",
		Type:                SiteTypeViafoura,
		BaseURL:             "https://lfpress.com",
		SitemapURL:          "https://lfpress.com/sitemap.xml",
		ViafouraSectionUUID: "00000000-0000-4000-8000-02ac0286cb1f",
	},
}
