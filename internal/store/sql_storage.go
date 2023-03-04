package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/salt-today/salttoday2/internal/sdk"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

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
		sdk.Logger(ctx).WithError(err).Error("Error shutting down SQL storage")
	}()

	return s, nil
}

func (s *sqlStorage) AddComments(comments ...*Comment) error {
	ds := s.dialect.Insert(CommentsTable).Cols(CommentsID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes)
	for _, comment := range comments {
		ds = ds.Vals(goqu.Vals{comment.ID, comment.UserID, comment.Time.Truncate(time.Second), comment.Text, comment.Likes, comment.Dislikes})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *sqlStorage) GetUserComments(userID int) ([]*Comment, error) {
	sd := s.dialect.
		Select(CommentsID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes).
		From(CommentsTable).
		Where(goqu.Ex{CommentsUserID: userID})

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*Comment, 0)
	var id, foundUserID int
	var tm time.Time
	var text string
	var likes, dislikes int32
	for rows.Next() {
		err := rows.Scan(&id, &foundUserID, &tm, &text, &likes, &dislikes)
		if err != nil {
			return nil, err
		}

		comments = append(comments, &Comment{
			ID:       id,
			UserID:   foundUserID,
			Time:     tm.Local(),
			Text:     text,
			Likes:    likes,
			Dislikes: dislikes,
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

func (s *sqlStorage) AddArticles(articles ...*Article) error {
	ds := s.dialect.Insert(ArticlesTable).Cols(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime)
	for _, article := range articles {
		ds = ds.Vals(goqu.Vals{article.ID, article.Url, article.Title, article.DiscoveryTime})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *sqlStorage) AddUsers(users ...*User) error {
	ds := goqu.Insert(UsersTable).Cols(UsersID, UsersName).OnConflict(exp.NewDoUpdateConflictExpression(UsersID, "REPLACE"))
	for _, user := range users {
		ds = ds.Vals(goqu.Vals{user.ID, user.UserName})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *sqlStorage) GetUnscrapedArticlesSince(scrapeThreshold time.Time) ([]*Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Or(
				goqu.Ex{
					ArticlesLastScrapeTime: nil,
				},
				goqu.Ex{
					ArticlesLastScrapeTime: goqu.Op{"lt": scrapeThreshold.Truncate(time.Second)},
				},
			),
		)

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *sqlStorage) SetArticleScrapedNow(articleIDs ...int) error {
	ds := s.dialect.Update(ArticlesTable).
		Where(goqu.Ex{ArticlesID: articleIDs}).
		Set(goqu.Record{ArticlesLastScrapeTime: time.Now().Truncate(time.Second)})

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *sqlStorage) SetArticleScrapedAt(scrapedTime time.Time, articleIDs ...int) error {
	ds := s.dialect.Update(ArticlesTable).
		Where(goqu.Ex{ArticlesID: articleIDs}).
		Set(goqu.Record{ArticlesLastScrapeTime: scrapedTime.Truncate(time.Second)})

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *sqlStorage) shutdown() error {
	return s.db.Close()
}
