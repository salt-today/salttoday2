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

// TODO uncomment sites when we're ready, no need to hammer all of them yet
// List taken from here: villagemedia.ca/sites/
var SitesMap = map[string]string{
	"TBNewsWatch": "https://www.tbnewswatch.com",
	// "BarrieToday":   "https://www.barrietoday.com",
	"BayToday": "https://www.baytoday.ca",
	// "BradfordToday": "https://www.bradfordtoday.ca",
	// "BurlingtonToday":  "https://burlingtontoday.com",
	// "CambridgeToday":   "https://www.cambridgetoday.ca",
	// "CollingwoodToday": "https://www.collingwoodtoday.ca",
	// "ElliotLakeToday":  "https://www.elliotlaketoday.com",
	// "EloraFergusToday": "https://www.elorafergustoday.com",
	"GuelphToday": "https://www.guelphtoday.com",
	// "HaltonHillsToday": "https://www.haltonhillstoday.ca",
	// "innisfilToday":           "https://www.innisfiltoday.ca",
	// "MidlandToday":            "https://midlandtoday.ca",
	// "NewMarketToday": "https://www.newmarkettoday.ca",
	// "NotLocal":                "https://www.notllocal.com",
	// "NorthernOntarioBusiness": "https://www.northernontariobusiness.com",
	"OrilliaMatters": "https://www.orilliamatters.com",
	// "PelhamToday":             "https://www.pelhamtoday.ca",
	"SooToday": "https://www.sootoday.com",
	// "StratfordToday": "https://www.stratfordtoday.ca",
	// "Sudbury":         "https://sudbury.com", // This requires Javascript to be enabled. The jerks.
	// "ThoroldToday":            "https://www.thoroldtoday.ca",
	// "TimminsToday": "https://www.timminstoday.com",
	// "AlimoshoToday": "https://www.alimoshotoday.com",
	// "BroomfieldLeader":        "https://www.broomfieldleader.com",
	// "Lasutoday":      "https://www.lasutoday.com",
	// "LongmontLeader": "https://www.longmontleader.com",
	// "Sooleader":              "https://www.sooleader.com",
	// "BkReader":               "https://www.bkreader.com",            // verify this works, doesn't look like others
	// "CharlestonCityPaper": "https://www.charlestoncitypaper.com", // verify this works, doesn't look like others
	// "NowNewsWatch":           "https://www.nwonewswatch.com",        // I think this one is just articles from the other sites in the region
	// "SNNewsWatch": "https://www.snnewswatch.com", // I think this one is just articles from the other sites in the region
	// "ChulavisToday":          "https://chulavistatoday.com",
	// "LivermoreVine":          "https://www.livermorevine.com", // verify this works, doesn't look like others
	// "RWCPulse":               "https://www.rwcpulse.com",      // verify this works, doesn't look like others
	// "AlaskaHighwayNews":      "https://www.alaskahighwaynews.ca",
	// "BowenIslandUnderCurrent": "https://www.bowenislandundercurrent.com",
	// "BurnabyNow":             "https://www.burnabynow.com",
	// "CoastReporter": "https://www.coastreporter.net",
	// "DawsonCreekMirror":       "https://www.dawsoncreekmirror.ca",
	// "DeltaOptimist":       "https://www.delta-optimist.com",
	// "TheReminder": "https://www.thereminder.ca",
	// "KamloopsThisWeek": "https://www.kamloopsthisweek.com",
	// "MooseJawToday": "https://www.moosejawtoday.com",
	// "NewwestRecord":       "https://newwestrecord.ca",
	// "NSNews":             "https://www.nsnews.com",
	// "PiqeueNewsMagazine": "https://www.piquenewsmagazine.com",
	// "PRPeak":             "https://www.prpeak.com",
	// "Praireag":            "https://www.prairieag.com",
	"PrinceGeorgeCitizen": "https://www.princegeorgecitizen.com",
	// "RichmondNews":        "https://www.richmond-news.com",
	// "SaskToday":           "https://www.sasktoday.ca", // verify this works, doesn't look like others
	// "SquamishChief": "https://www.squamishchief.com",
	// "ThompsonCitizen":     "https://www.thompsoncitizen.net",
	// "TricityNews":        "https://www.tricitynews.com",
	// "VancouverIsAwesome": "https://www.vancouverisawesome.com",
	// "TimesColonist":      "https://www.timescolonist.com",
	// "EmpireAdvance":      "https://www.empireadvance.ca",
	// "WesternInvestor":     "https://www.westerninvestor.com",
	// 	"AirdrieToday":      "https://www.airdrietoday.com",
	// "AlbertaPrimeTimes": "https://www.albertaprimetimes.com",
	// "CochraneToday":     "https://www.cochranetoday.ca",
	// "LakelandToday":     "https://www.lakelandtoday.ca",
	// "MountainviewToday": "https://www.mountainviewtoday.ca",
	"OkotoksToday": "https://www.okotokstoday.ca",
	// "RMOutlook":           "https://www.rmoutlook.com",
	// "StalbertGazette":     "https://www.stalbertgazette.com",
	// "TownAndCountryToday": "https://www.townandcountrytoday.com",
	// "GriceConnect":        "https://www.griceconnect.com",
	// "LocalProfile": "https://www.localprofile.com", // verify this works, doesn't look like others
	// "GazetteLeader":       "https://www.gazetteleader.com",
	// "QueenCreekSunTimes": "https://www.queencreeksuntimes.com",
	// "RoughDraftAtlanta":  "https://www.roughdraftatlanta.com", // verify this works, doesn't look like others
	// "HalifaxCityNews":     "https://www.halifax.citynews.ca",   // verify this works, doesn't look like others
	// "KitchenerCityNews":   "https://www.kitchener.citynews.ca", // verify this works, doesn't look like others
	// "OttawaCityNews":          "https://www.ottawa.citynews.ca",      // verify this works, doesn't look like others
	// "WashingtonCityPaper": "https://www.washingtoncitypaper.com", // verify this works, doesn't look like others
}
