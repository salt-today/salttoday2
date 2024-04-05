package rdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb/migrations"
)

var _ store.Storage = (*sqlStorage)(nil)

type sqlStorage struct {
	db      *sql.DB
	dialect goqu.DialectWrapper
}

const maxPageSize uint = 20

func getSqlConnString(ctx context.Context) string {
	url := os.Getenv("MYSQL_URL")

	if url == `` {
		sdk.Logger(ctx).Info("Missing database configuration, defaulting to local dev")
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
	}

	go func() {
		<-ctx.Done()
		err := s.shutdown()
		if err != nil {
			sdk.Logger(ctx).WithError(err).Error("Error shutting down SQL storage")
		}
	}()

	return s, nil
}

func (s *sqlStorage) AddComments(ctx context.Context, comments ...*store.Comment) error {
	ds := s.dialect.Insert(CommentsTable).
		Cols(CommentsID, CommentsArticleID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes).
		As(NewAlias).
		OnConflict(goqu.DoUpdate(CommentsLikes, goqu.C(LikesSuffix).Set(goqu.I(NewAliasLikes)))).
		OnConflict(goqu.DoUpdate(CommentsDislikes, goqu.C(DislikesSuffix).Set(goqu.I(NewAliasDislikes))))

	for _, comment := range comments {
		ds = ds.Vals(goqu.Vals{comment.ID, comment.Article.ID, comment.User.ID, comment.Time.Truncate(time.Second), comment.Text, comment.Likes, comment.Dislikes})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetUsersStats(ctx context.Context, opts store.UserQueryOptions) ([]*store.UserStats, error) {
	sd := s.dialect.
		Select(UsersID, UsersName, goqu.SUM(CommentsLikes).As(UserLikes), goqu.SUM(CommentsDislikes).As(UserDislikes)).
		From(UsersTable).
		InnerJoin(goqu.T(CommentsTable).As(CommentsTable), goqu.On(goqu.I(UsersID).Eq(goqu.I(CommentsUserID)))).
		GroupBy(UsersID)

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{UsersID: opts.ID})
	}

	sd = addPaging(sd, &opts.PageOpts)

	// TODO Implement other options
	// site, order

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*store.UserStats, 0)

	for rows.Next() {
		sdk.Logger(ctx).Info("scanning user")
		u := &store.UserStats{
			User: &store.User{},
		}
		err := rows.Scan(&u.User.ID, &u.User.UserName, &u.TotalLikes, &u.TotalDislikes)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user record: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}

func (s *sqlStorage) GetComments(ctx context.Context, opts store.CommentQueryOptions) ([]*store.Comment, error) {
	sd := s.dialect.
		Select(CommentsID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes, goqu.L(CommentsLikes+" + 2 * "+CommentsDislikes).
			As(CommentsScore), CommentsDeleted, CommentsArticleID, CommentsUserID, ArticlesID, ArticlesTitle, ArticlesUrl, UsersID, UsersName).
		From(CommentsTable).
		InnerJoin(goqu.T(UsersTable).As(UsersTable), goqu.On(goqu.I(CommentsUserID).Eq(goqu.I(UsersID)))).
		InnerJoin(goqu.T(ArticlesTable).As(ArticlesTable), goqu.On(goqu.I(CommentsArticleID).Eq(goqu.I(ArticlesID))))

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{CommentsID: opts.ID})
	}

	if opts.UserID != nil {
		sd = sd.Where(goqu.Ex{CommentsUserID: opts.UserID})
	}

	if opts.PageOpts.Site != nil {
		// TODO
		// sd = sd.Where()
	}

	if opts.OnlyDeleted == true {
		sd = sd.Where(goqu.Ex{CommentsDeleted: true})
	}

	if opts.DaysAgo != nil && *opts.DaysAgo != 0 {
		sd = sd.Where(goqu.I(CommentsTime).Gt(goqu.L("NOW() - INTERVAL ? DAY", *opts.DaysAgo)))
	}

	sd = addPaging(sd, &opts.PageOpts)

	if opts.PageOpts.Order != nil {
		if *opts.PageOpts.Order == store.OrderByLikes {
			sd = sd.Order(goqu.I(CommentsLikes).Desc())
		} else if *opts.PageOpts.Order == store.OrderByDislikes {
			sd = sd.Order(goqu.I(CommentsDislikes).Desc())
		} else if *opts.PageOpts.Order == store.OrderByBoth {
			sd = sd.Order(goqu.I(CommentsScore).Desc())
		} else {
			return nil, fmt.Errorf("unexpected ordering directive %d", *opts.PageOpts.Order)
		}
	}

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*store.Comment, 0)
	var id, articleID, commentsArticleID, userID, commentsUserID int
	var tm time.Time
	var text string
	var likes, dislikes int32
	var score int64
	var deleted bool
	var articleTitle string
	var articleUrl string
	var userName string
	for rows.Next() {
		err := rows.Scan(&id, &tm, &text, &likes, &dislikes, &score, &deleted, &commentsArticleID, &commentsUserID, &articleID, &articleTitle, &articleUrl, &userID, &userName)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &store.Comment{
			ID:       id,
			Article:  store.Article{ID: articleID, Title: articleTitle, Url: articleUrl},
			User:     store.User{ID: userID, UserName: userName},
			Time:     tm.Local(),
			Text:     text,
			Likes:    likes,
			Dislikes: dislikes,
			Deleted:  deleted,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return nil, &store.NoQueryResultsError{}
	}

	return comments, nil
}

func (s *sqlStorage) AddArticles(ctx context.Context, articles ...*store.Article) error {
	ds := s.dialect.Insert(ArticlesTable).Cols(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		OnConflict(goqu.DoNothing())

	// We want to set the lastScrapedTime to nil so that the article will be scraped immediately
	for _, article := range articles {
		ds = ds.Vals(goqu.Vals{article.ID, article.Url, article.Title, article.DiscoveryTime, nil})
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
	ds := s.dialect.Insert(UsersTable).Cols(UsersID, UsersName).OnConflict(goqu.DoNothing()) // TODO Something other than nothing
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

func (s *sqlStorage) GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Or(
				goqu.Ex{
					ArticlesLastScrapeTime: nil,
				},
				goqu.Ex{
					ArticlesDiscoveryTime: goqu.Op{"lte": scrapeThreshold.Truncate(time.Second)},
				},
			),
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

func (s *sqlStorage) GetRecentlyDiscoveredArticles(ctx context.Context, threshold time.Time) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Ex{
				ArticlesLastScrapeTime: goqu.Op{"gte": threshold.Truncate(time.Second)},
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

func (s *sqlStorage) SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error {
	ds := s.dialect.Update(ArticlesTable).
		Where(goqu.Ex{ArticlesID: articleIDs}).
		Set(goqu.Record{ArticlesLastScrapeTime: time.Now().Truncate(time.Second)})

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
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

func (s *sqlStorage) shutdown() error {
	return s.db.Close()
}
