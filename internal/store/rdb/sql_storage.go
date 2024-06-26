package rdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb/migrations"
)

const (
	maxPageSize uint = 20
)

var _ store.Storage = (*sqlStorage)(nil)

type sqlStorage struct {
	db      *sql.DB
	dialect goqu.DialectWrapper

	cachedResults *cachedResults
}

// Store expensive results here
type cachedResults struct {
	topScoringUser  map[string]*store.User
	topLikedUser    map[string]*store.User
	topDislikedUser map[string]*store.User
	topScoringSite  *store.Site
	topLikedSite    *store.Site
	topDislikedSite *store.Site
}

func getSqlConnString(ctx context.Context) string {
	url := os.Getenv("MYSQL_URL")

	if url == `` {
		logger.New(ctx).Info("Missing database configuration, defaulting to local dev")
		return "root:salt@tcp(localhost:3306)/salt"
	}
	return url
}

func New(ctx context.Context) (*sqlStorage, error) {
	entry := logrus.WithField(`component`, `sql-storage`)

	db, err := sql.Open("mysql", getSqlConnString(ctx)+"?parseTime=true")
	if err != nil {
		return nil, err
	}
	entry.Info("successfully connected to database")

	err = migrations.MigrateDb(db)
	if err != nil {
		return nil, err
	}

	s := &sqlStorage{
		db:      db,
		dialect: goqu.Dialect("mysql"),
		cachedResults: &cachedResults{
			topScoringUser:  make(map[string]*store.User),
			topLikedUser:    make(map[string]*store.User),
			topDislikedUser: make(map[string]*store.User),
		},
	}

	// periodically calculate the most likes, dislikes
	err = s.cacheTopResults(ctx)
	if err != nil {
		entry.Error("Unable to cache top results on storage startup")
		return nil, err
	}

	ticker := time.NewTicker(time.Hour)
	go func() error {
		<-ticker.C
		return s.cacheTopSites(ctx)
	}()

	go func() {
		<-ctx.Done()
		err := s.shutdown()
		if err != nil {
			logger.New(ctx).WithError(err).Error("Error shutting down SQL storage")
		}
	}()

	// err = s.fillSiteName(ctx)
	return s, err
}

func (s *sqlStorage) fillSiteName(ctx context.Context) error {
	for siteName, siteUrl := range internal.SitesMap {
		ds := s.dialect.Update(ArticlesTable).
			Set(goqu.Record{ArticlesSiteName: siteName}).
			Where(goqu.I(ArticlesUrl).Like(siteUrl + "%"))

		query, _, err := ds.ToSQL()
		if err != nil {
			return err
		}
		_, err = s.db.ExecContext(ctx, query)
		if err != nil {
			return err
		}

	}
	logger.New(ctx).Info("Successfully updated site names")
	return nil
}

func (s *sqlStorage) cacheTopResults(ctx context.Context) error {
	err := s.cacheTopSites(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *sqlStorage) GetTopSite(ctx context.Context, orderBy int) (*store.Site, error) {
	if orderBy == store.OrderByBoth {
		if s.cachedResults.topScoringSite == nil {
			return nil, &store.NoQueryResultsError{}
		}
		return s.cachedResults.topScoringSite, nil

	} else if orderBy == store.OrderByLikes {
		if s.cachedResults.topLikedSite == nil {
			return nil, &store.NoQueryResultsError{}
		}
		return s.cachedResults.topLikedSite, nil

	} else if orderBy == store.OrderByDislikes {
		if s.cachedResults.topDislikedSite == nil {
			return nil, &store.NoQueryResultsError{}
		}
		return s.cachedResults.topDislikedSite, nil

	} else {
		return nil, fmt.Errorf("unknown orderBy %d", orderBy)
	}
}

func (s *sqlStorage) cacheTopSites(ctx context.Context) error {
	entry := logger.New(ctx)

	opts := &store.PageQueryOptions{
		Order: store.OrderByBoth,
		Limit: aws.Uint(1),
	}

	sites, err := s.GetSites(ctx, opts)
	if err != nil {
		entry.WithError(err).Error("unable to calculate highest scoring site")
		return err
	} else if len(sites) < 1 {
		entry.Warn("no sites found")
		return nil
	}
	s.cachedResults.topScoringSite = sites[0]

	opts.Order = store.OrderByLikes
	sites, err = s.GetSites(ctx, opts)
	if err != nil {
		entry.WithError(err).Error("unable to calculate highest liked site")
	}
	s.cachedResults.topLikedSite = sites[0]

	opts.Order = store.OrderByDislikes
	sites, err = s.GetSites(ctx, opts)
	if err != nil {
		entry.WithError(err).Error("unable to calculate highest disliked site")
		return err
	}
	s.cachedResults.topDislikedSite = sites[0]

	return nil
}

func (s *sqlStorage) AddComments(ctx context.Context, comments []*store.Comment) error {
	// We need to add comments aritlce by article so we can easily determine if a comment was deleted or not
	articleCommentsMap := make(map[int][]*store.Comment)
	for _, comment := range comments {
		articleCommentsMap[comment.Article.ID] = append(articleCommentsMap[comment.Article.ID], comment)
	}

	var err error
	for articleID, comments := range articleCommentsMap {
		addErr := s.addCommentsToArticle(ctx, articleID, comments)
		err = errors.Join(err, addErr)
	}

	return err
}

func (s *sqlStorage) addCommentsToArticle(ctx context.Context, articleID int, comments []*store.Comment) error {
	entry := logger.New(ctx).WithField("articleID", articleID)

	// Determine if any comments were deleted
	queryOpts := &store.CommentQueryOptions{
		ArticleID: &articleID,
		PageOpts: &store.PageQueryOptions{
			Order: store.OrderByBoth,
		},
	}

	storedComments, err := s.GetComments(ctx, queryOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		// no-op
		entry.Info("New article, no comments found")
	} else if err != nil {
		entry.WithError(err).Error("Unable to get comments while adding new comments, required for determining if comments are deleted")
		return err
	}

	commentsMap := make(map[int]*store.Comment)
	for _, comment := range comments {
		commentsMap[comment.ID] = comment
	}

	for _, storedComment := range storedComments {
		if _, ok := commentsMap[storedComment.ID]; !ok {
			if !storedComment.Deleted {
				entry.WithField("commentID", storedComment.ID).Info("Found comment was deleted!")
				storedComment.Deleted = true
				comments = append(comments, storedComment)
			}
		} else {
			// Update just incase we found a comment that we thought was deleted before, but we're just bad at scraping
			commentsMap[storedComment.ID].Deleted = false
		}
	}

	// Upsert comment into database
	ds := s.dialect.Insert(CommentsTable).
		Cols(CommentsID, CommentsArticleID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes, CommentsDeleted).
		As(NewAlias).
		OnConflict(goqu.DoUpdate(OldAlias, goqu.Record{
			CommentsLikes:    goqu.I(NewAliasLikes),
			CommentsDislikes: goqu.I(NewAliasDislikes),
			CommentsDeleted:  goqu.I(NewAliasDeleted),
		}))

	for _, comment := range comments {
		ds = ds.Vals(goqu.Vals{comment.ID, comment.Article.ID, comment.User.ID, comment.Time.Truncate(time.Second), comment.Text, comment.Likes, comment.Dislikes, comment.Deleted})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)

	return err
}

func (s *sqlStorage) GetUsers(ctx context.Context, opts *store.UserQueryOptions) ([]*store.User, error) {
	sd := s.dialect.
		From(UsersTable).
		InnerJoin(goqu.T(CommentsTable).As(CommentsTable), goqu.On(goqu.I(UsersID).Eq(goqu.I(CommentsUserID)))).
		GroupBy(UsersID)

	// only get the comments we need since we're summing all the values
	cols := []interface{}{UsersID, UsersName}
	if opts.PageOpts.Order == store.OrderByLikes {
		cols = append(cols, goqu.SUM(CommentsLikes).As(UserLikes))
		sd = sd.Order(goqu.I(UserLikes).Desc())
	} else if opts.PageOpts.Order == store.OrderByDislikes {
		cols = append(cols, goqu.SUM(CommentsDislikes).As(UserDislikes))
		sd = sd.Order(goqu.I(UserDislikes).Desc())
	} else {
		cols = append(cols, goqu.SUM(CommentsLikes).As(UserLikes), goqu.SUM(CommentsDislikes).As(UserDislikes))
		sd = sd.Order(goqu.L(UserLikes + "+" + UserDislikes).Desc())
	}
	sd = sd.Select(cols...)

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{UsersID: opts.ID})
	}

	if opts.Name != `` {
		// Get users where their name contains opts.Name and ignoring case
		sd = sd.Where(goqu.I(UsersName).ILike("%" + opts.Name + "%"))
	}

	if opts.PageOpts.Site != `` {
		sd = sd.Where(goqu.I(ArticlesSiteName).Eq(opts.PageOpts.Site)).
			InnerJoin(goqu.T(ArticlesTable).As(ArticlesTable), goqu.On(goqu.I(CommentsArticleID).Eq(goqu.I(ArticlesID))))
	}

	sd = addPaging(sd, opts.PageOpts)

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*store.User

	for rows.Next() {
		u := &store.User{}
		dests := []interface{}{&u.ID, &u.UserName}
		if opts.PageOpts.Order == store.OrderByLikes {
			dests = append(dests, &u.TotalLikes)
		} else if opts.PageOpts.Order == store.OrderByDislikes {
			dests = append(dests, &u.TotalDislikes)
		} else {
			dests = append(dests, &u.TotalLikes, &u.TotalDislikes)
		}
		err := rows.Scan(dests...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user record: %w", err)
		}

		// TODO feels bad.
		// Have to calculate score since it's not calculated in select anymore
		u.TotalScore = u.TotalDislikes + u.TotalLikes
		users = append(users, u)
	}
	return users, nil
}

func (s *sqlStorage) GetComments(ctx context.Context, opts *store.CommentQueryOptions) ([]*store.Comment, error) {
	cols := []interface{}{
		CommentsID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes,
		CommentsDeleted, ArticlesID, ArticlesTitle, ArticlesSiteName, ArticlesUrl, UsersID, UsersName,
	}
	sd := s.dialect.
		From(CommentsTable).
		InnerJoin(goqu.T(UsersTable).As(UsersTable), goqu.On(goqu.I(CommentsUserID).Eq(goqu.I(UsersID)))).
		InnerJoin(goqu.T(ArticlesTable).As(ArticlesTable), goqu.On(goqu.I(CommentsArticleID).Eq(goqu.I(ArticlesID))))

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{CommentsID: opts.ID})
	}

	if opts.UserID != nil {
		sd = sd.Where(goqu.Ex{CommentsUserID: opts.UserID})
	}

	if opts.PageOpts.Site != `` {
		sd = sd.Where(goqu.I(ArticlesSiteName).Eq(opts.PageOpts.Site))
	}

	if opts.OnlyDeleted {
		sd = sd.Where(goqu.Ex{CommentsDeleted: true})
	}

	if opts.DaysAgo != 0 {
		sd = sd.Where(goqu.I(CommentsTime).Gt(goqu.L("NOW() - INTERVAL ? DAY", opts.DaysAgo)))
	}

	if opts.ArticleID != nil {
		sd = sd.Where(goqu.Ex{CommentsArticleID: *opts.ArticleID})
	}

	if opts.Text != `` {
		sd = sd.Where(goqu.I(CommentsText).ILike("%" + opts.Text + "%"))
	}

	sd = addPaging(sd, opts.PageOpts)

	if opts.PageOpts.Order == store.OrderByLikes {
		sd = sd.Order(goqu.I(CommentsLikes).Desc())
	} else if opts.PageOpts.Order == store.OrderByDislikes {
		sd = sd.Order(goqu.I(CommentsDislikes).Desc())
	} else if opts.PageOpts.Order == store.OrderByBoth {
		cols = append(cols, goqu.L(CommentsLikes+" + "+CommentsDislikes).As(CommentsScore))
		sd = sd.Order(goqu.I(CommentsScore).Desc())
	} else if opts.PageOpts.Order == store.OrderByControversial {
		cols = append(cols, CommentControverstyWeightedEntropy)
		sd = sd.Order(goqu.I(CommentControverstyWeightedEntropy).Desc()).
			InnerJoin(goqu.T(CommentControverstyView).As(CommentControverstyView), goqu.On(goqu.I(CommentsID).Eq(goqu.I(CommentControverstyID))))
	} else {
		return nil, fmt.Errorf("unexpected ordering directive %d", opts.PageOpts.Order)
	}

	sd = sd.Select(cols...)

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*store.Comment
	var score int
	var weightedEntropy float64
	for rows.Next() {
		c := &store.Comment{Article: store.Article{}, User: store.User{}}
		dests := []interface{}{&c.ID, &c.Time, &c.Text, &c.Likes, &c.Dislikes, &c.Deleted, &c.Article.ID, &c.Article.Title, &c.Article.SiteName, &c.Article.Url, &c.User.ID, &c.User.UserName}
		// no-op
		if opts.PageOpts.Order == store.OrderByBoth {
			dests = append(dests, &score)
		} else if opts.PageOpts.Order == store.OrderByControversial {
			dests = append(dests, &weightedEntropy)
		}
		err := rows.Scan(dests...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment record: %w", err)
		}

		comments = append(comments, c)
	}

	if len(comments) == 0 {
		return nil, &store.NoQueryResultsError{}
	}

	return comments, nil
}

func (s *sqlStorage) AddArticles(ctx context.Context, articles ...*store.Article) error {
	ds := s.dialect.Insert(ArticlesTable).Cols(ArticlesID, ArticlesSiteName, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		As(NewAlias).
		OnConflict(
			goqu.DoUpdate(ArticlesSiteName,
				goqu.C(SiteNameSuffix).Set(goqu.L("IF(?, ?, ?)", goqu.I(NewAliasSiteName).Eq(internal.AllSitesName), goqu.I(NewAliasSiteName), goqu.I(ArticlesSiteName)))))

	// We want to set the lastScrapedTime to nil so that the article will be scraped immediately
	for _, article := range articles {
		ds = ds.Vals(goqu.Vals{article.ID, article.SiteName, article.Url, article.Title, article.DiscoveryTime, nil})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetArticles(ctx context.Context, ids ...int) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(goqu.Ex{ArticlesID: ids})

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return hydrateArticles(rows)
}

func (s *sqlStorage) AddUsers(ctx context.Context, users ...*store.User) error {
	ds := s.dialect.Insert(UsersTable).Cols(UsersID, UsersName).OnConflict(goqu.DoNothing())
	for _, user := range users {
		ds = ds.Vals(goqu.Vals{user.ID, user.UserName})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetSites(ctx context.Context, opts *store.PageQueryOptions) ([]*store.Site, error) {
	sd := s.dialect.
		From(CommentsTable).
		InnerJoin(goqu.T(ArticlesTable), goqu.On(goqu.I(CommentsArticleID).Eq(goqu.I(ArticlesID)))).
		Where(goqu.Ex{ArticlesSiteName: goqu.Op{"neq": internal.AllSitesName}}).
		GroupBy(ArticlesSiteName)

	// only get the comments we need since we're summing all the values
	cols := []interface{}{ArticlesSiteName}
	if opts.Order == store.OrderByLikes {
		cols = append(cols, goqu.SUM(CommentsLikes).As(SiteLikes))
		sd = sd.Order(goqu.I(SiteLikes).Desc())
	} else if opts.Order == store.OrderByDislikes {
		cols = append(cols, goqu.SUM(CommentsDislikes).As(SiteDislikes))
		sd = sd.Order(goqu.I(SiteDislikes).Desc())
	} else {
		cols = append(cols, goqu.SUM(CommentsLikes).As(SiteLikes), goqu.SUM(CommentsDislikes).As(SiteDislikes))
		sd = sd.Order(goqu.L(SiteLikes + "+" + SiteDislikes).Desc())
	}
	sd = sd.Select(cols...)

	sd = addPaging(sd, opts)

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("unable to convert query to sql: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying db: %w", err)
	}
	defer rows.Close()

	var sites []*store.Site
	for rows.Next() {
		site := &store.Site{}
		dests := []interface{}{&site.Name}
		if opts.Order == store.OrderByLikes {
			dests = append(dests, &site.TotalLikes)
		} else if opts.Order == store.OrderByDislikes {
			dests = append(dests, &site.TotalDislikes)
		} else {
			dests = append(dests, &site.TotalLikes, &site.TotalDislikes)
		}
		err := rows.Scan(dests...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sites: %w", err)
		}

		// TODO feels bad.
		// Have to calculate score since it's not calculated in select anymore
		// TODO not convinced we actually have to do this anymore - verify
		site.TotalScore = site.TotalDislikes + site.TotalLikes
		sites = append(sites, site)
	}
	return sites, nil
}

func addPaging(sd *goqu.SelectDataset, pageOpts *store.PageQueryOptions) *goqu.SelectDataset {
	limit := maxPageSize
	if pageOpts.Limit != nil && *pageOpts.Limit < limit {
		limit = *pageOpts.Limit
	}
	sd = sd.Limit(limit)

	page := uint(0)
	if pageOpts.Page != nil {
		page = *pageOpts.Page
	}
	sd = sd.Offset(page * limit)
	return sd
}

func hydrateArticles(rows *sql.Rows) ([]*store.Article, error) {
	articles := make([]*store.Article, 0)
	var id int
	var url, title string
	var first, last sql.NullTime
	for rows.Next() {
		err := rows.Scan(&id, &url, &title, &first, &last)
		if err != nil {
			return nil, err
		}

		article := &store.Article{
			ID:            id,
			Url:           url,
			Title:         title,
			DiscoveryTime: first.Time.Local(),
		}

		if last.Valid {
			localTime := last.Time.Local()
			article.LastScrapeTime = localTime
		}

		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(articles) == 0 {
		return nil, &store.NoQueryResultsError{}
	}

	return articles, nil
}

func (s *sqlStorage) GetRecentlyDiscoveredArticles(ctx context.Context, threshold time.Time) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Ex{
				ArticlesDiscoveryTime: goqu.Op{"gte": threshold.Truncate(time.Second)},
			},
		)
	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles, err := hydrateArticles(rows)
	if err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *sqlStorage) SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error {
	ds := s.dialect.Update(ArticlesTable).
		Where(goqu.Ex{ArticlesID: articleIDs}).
		Set(goqu.Record{ArticlesLastScrapeTime: scrapedTime.Truncate(time.Second)})

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetStats(ctx context.Context) (*store.Stats, error) {
	stats := &store.Stats{}
	sd := s.dialect.Select(goqu.COUNT(CommentsID)).From(CommentsTable)
	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(ctx, query)
	err = row.Scan(&stats.CommentCount)
	if err != nil {
		return nil, fmt.Errorf("error counting comments %v", err)
	}

	sd = s.dialect.Select(goqu.COUNT(CommentsID)).From(CommentsTable).Where(goqu.Ex{CommentsDeleted: true})
	query, _, err = sd.ToSQL()
	if err != nil {
		return nil, err
	}

	row = s.db.QueryRowContext(ctx, query)
	err = row.Scan(&stats.DeletedCount)
	if err != nil {
		return nil, fmt.Errorf("error counting deleted comments %v", err)
	}

	sd = s.dialect.Select(goqu.SUM(CommentsLikes)).From(CommentsTable)
	query, _, err = sd.ToSQL()
	if err != nil {
		return nil, err
	}

	row = s.db.QueryRowContext(ctx, query)
	err = row.Scan(&stats.LikeCount)
	if err != nil {
		return nil, fmt.Errorf("error counting likes %v", err)
	}

	sd = s.dialect.Select(goqu.SUM(CommentsDislikes)).From(CommentsTable)
	query, _, err = sd.ToSQL()
	if err != nil {
		return nil, err
	}

	row = s.db.QueryRowContext(ctx, query)
	err = row.Scan(&stats.DislikeCount)
	if err != nil {
		return nil, fmt.Errorf("error counting dislikes %v", err)
	}

	sd = s.dialect.Select(goqu.COUNT(ArticlesID)).From(ArticlesTable)
	query, _, err = sd.ToSQL()
	if err != nil {
		return nil, err
	}
	row = s.db.QueryRowContext(ctx, query)
	err = row.Scan(&stats.ArticleCount)
	if err != nil {
		return nil, fmt.Errorf("error counting articles %v", err)
	}

	sd = s.dialect.Select(goqu.COUNT(UsersID)).From(UsersTable)
	query, _, err = sd.ToSQL()
	if err != nil {
		return nil, err
	}
	row = s.db.QueryRowContext(ctx, query)
	err = row.Scan(&stats.UserCount)
	if err != nil {
		return nil, fmt.Errorf("error counting users %v", err)
	}

	return stats, nil
}

func (s *sqlStorage) shutdown() error {
	return s.db.Close()
}
