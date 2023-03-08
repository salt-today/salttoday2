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

const maxPageSize uint = 20

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

func (s *sqlStorage) GetComments(ctx context.Context, opts CommentQueryOptions) ([]*Comment, error) {
	sd := s.dialect.
		Select(CommentsID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes, goqu.L(CommentsLikes+" + "+CommentsDislikes).As(CommentsScore), CommentsDeleted, CommentsArticleID, CommentsUserID).
		From(CommentsTable)

	limit := maxPageSize
	if opts.Limit != nil && *opts.Limit < limit {
		limit = *opts.Limit
	}
	sd = sd.Limit(limit)

	if opts.Order != nil {
		if *opts.Order == OrderByLiked {
			sd = sd.Order(goqu.I(CommentsLikes).Desc())
		} else if *opts.Order == OrderByDisliked {
			sd = sd.Order(goqu.I(CommentsDislikes).Desc())
		} else if *opts.Order == OrderByBoth {
			sd = sd.Order(goqu.I(CommentsScore).Desc())
		} else {
			return nil, fmt.Errorf("unexpected ordering directive %d", *opts.Order)
		}
	}

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{CommentsID: opts.ID})
	}

	if opts.UserID != nil {
		sd = sd.Where(goqu.Ex{CommentsUserID: opts.UserID})
	}

	if opts.Site != nil {
		// TODO
		// sd = sd.Where()
	}

	if opts.OnlyDeleted == true {
		sd = sd.Where(goqu.Ex{CommentsDeleted: true})
	}

	if opts.DaysAgo != nil {
		sd = sd.Where(goqu.I(CommentsTime).Gt(goqu.L("NOW() - INTERVAL ? DAY", *opts.DaysAgo)))
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
	var score int64
	var deleted bool
	for rows.Next() {
		err := rows.Scan(&id, &tm, &text, &likes, &dislikes, &score, &deleted, &articleID, &foundUserID)
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

func (s *sqlStorage) GetArticles(ctx context.Context, ids ...int) ([]*Article, error) {
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

func (s *sqlStorage) AddUsers(ctx context.Context, users ...*User) error {
	// TODO: this conflict expression isn't valid and errors
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

func (s *sqlStorage) GetUsersByIDs(ctx context.Context, ids ...int) ([]*User, error) {
	sd := s.dialect.Select(UsersID, UsersName).
		From(UsersTable).
		Where(goqu.Ex{UsersID: ids})

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*User, 0)
	var id int
	var name string
	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			return nil, err
		}

		users = append(users, &User{
			ID:       id,
			UserName: name,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *sqlStorage) GetUserByName(ctx context.Context, name string) (*User, error) {
	sd := s.dialect.Select(UsersID, UsersName).
		From(UsersTable).
		Where(goqu.Ex{UsersName: name})

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int
	var foundName string
	if rows.Next() {
		err := rows.Scan(&id, &foundName)
		if err != nil {
			return nil, err
		}
		return &User{
			ID:       id,
			UserName: foundName,
		}, nil
	}
	// TODO return error if not found?
	return nil, nil
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
