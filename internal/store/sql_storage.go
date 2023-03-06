package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/salt-today/salttoday2/internal/sdk"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/doug-martin/goqu/v9/exp"
	_ "github.com/go-sql-driver/mysql"
)

var _ Storage = (*sqlStorage)(nil)

type sqlStorage struct {
	db      *sql.DB
	dialect goqu.DialectWrapper
}

func NewSQLStorage(ctx context.Context) (*sqlStorage, error) {
	// TODO: conn string should be configurable
	db, err := sql.Open("mysql", "salt:salt@tcp(localhost:3306)/salt?parseTime=true")
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

func (s *sqlStorage) AddComments(ctx context.Context, comments ...*Comment) error {
	ds := s.dialect.Insert(CommentsTable).Cols(CommentsID, CommentsArticleID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes)
	for _, comment := range comments {
		ds = ds.Vals(goqu.Vals{comment.ID, comment.ArticleID, comment.UserID, comment.Time.Truncate(time.Second), comment.Text, comment.Likes, comment.Dislikes})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetUserComments(ctx context.Context, userID int, opts QueryOptions) ([]*Comment, error) {
	sd := s.dialect.
		Select(CommentsID, CommentsArticleID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes).
		From(CommentsTable).
		Where(goqu.Ex{CommentsUserID: userID})

	// TODO Process City and ShowDeleted Parameters
	if opts.Limit != nil {
		sd = sd.Limit(*opts.Limit)
	}
	if opts.Order != nil {
		if *opts.Order == OrderByLiked {
			sd = sd.Order(goqu.I(CommentsLikes).Desc())
		} else if *opts.Order == OrderByDisliked {
			sd = sd.Order(goqu.I(CommentsDislikes).Desc())
		} else {
			return nil, fmt.Errorf("unexpected ordering directive %d", *opts.Order)
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

	comments := make([]*Comment, 0)
	var id, articleID, foundUserID int
	var tm time.Time
	var text string
	var likes, dislikes int32
	for rows.Next() {
		err := rows.Scan(&id, &articleID, &foundUserID, &tm, &text, &likes, &dislikes)
		if err != nil {
			return nil, err
		}

		comments = append(comments, &Comment{
			ID:        id,
			ArticleID: articleID,
			UserID:    foundUserID,
			Time:      tm.Local(),
			Text:      text,
			Likes:     likes,
			Dislikes:  dislikes,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (s *sqlStorage) AddArticles(ctx context.Context, articles ...*Article) error {
	ds := s.dialect.Insert(ArticlesTable).Cols(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime)
	for _, article := range articles {
		ds = ds.Vals(goqu.Vals{article.ID, article.Url, article.Title, article.DiscoveryTime})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) AddUsers(ctx context.Context, users ...*User) error {
	ds := goqu.Insert(UsersTable).Cols(UsersID, UsersName).OnConflict(exp.NewDoUpdateConflictExpression(UsersID, "REPLACE"))
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

func hydrateArticles(rows *sql.Rows) ([]*Article, error) {
	articles := make([]*Article, 0)
	var id int
	var url, title string
	var first, last sql.NullTime
	for rows.Next() {
		err := rows.Scan(&id, &url, &title, &first, &last)
		if err != nil {
			return nil, err
		}

		article := &Article{
			ID:            id,
			Url:           url,
			Title:         title,
			DiscoveryTime: first.Time.Local(),
		}

		if last.Valid {
			localTime := last.Time.Local()
			article.LastScrapeTime = &localTime
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *sqlStorage) GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Or(
				goqu.Ex{
					ArticlesLastScrapeTime: nil,
				},
				goqu.Ex{
					ArticlesLastScrapeTime: goqu.Op{"lte": scrapeThreshold.Truncate(time.Second)},
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

func (s *sqlStorage) GetRecentlyDiscoveredArticles(ctx context.Context, threshold time.Time) ([]*Article, error) {
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
