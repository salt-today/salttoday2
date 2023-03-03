package store

import (
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type storage struct {
	db *sql.DB
}

func NewStorage() (*storage, error) {
	// TODO: conn string should be configurable
	db, err := sql.Open("mysql", "salt:salt@tcp(localhost:3306)/salt?parseTime=true")
	if err != nil {
		return nil, err
	}

	return &storage{
		db: db,
	}, nil
}

func (s *storage) AddComments(comments ...*Comment) error {
	ds := goqu.Dialect("mysql").Insert(CommentsTable).Cols(CommentsUser, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes)
	for _, comment := range comments {
		ds = ds.Vals(goqu.Vals{comment.Name, comment.Time.Truncate(time.Second), comment.Text, comment.Likes, comment.Dislikes})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *storage) GetUserComments(user string) ([]*Comment, error) {
	sd := goqu.Dialect("mysql").
		Select(CommentsUser, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes).
		From(CommentsTable).
		Where(goqu.Ex{CommentsUser: user})

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
	var foundUser string
	var tm time.Time
	var text string
	var likes int32
	var dislikes int32
	for rows.Next() {
		err := rows.Scan(&foundUser, &tm, &text, &likes, &dislikes)
		if err != nil {
			return nil, err
		}

		comments = append(comments, &Comment{
			Name:     foundUser,
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

func (s *storage) AddArticles(articles ...*Article) error {
	ds := goqu.Insert(ArticlesTable).Cols(ArticlesUrl, ArticlesTitle)
	for _, article := range articles {
		ds = ds.Vals(goqu.Vals{article.Url, article.Title})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *storage) AddUsers(users ...string) error {
	ds := goqu.Insert(UsersTable).Cols(UsersUser)
	for _, user := range users {
		ds = ds.Vals(goqu.Vals{user})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(query)
	return err
}

func (s *storage) GetArticleIDsSince(time.Time) ([]int, error) {
	// TODO implement him
	panic("implement me")
}

func (s *storage) Shutdown() error {
	return s.db.Close()
}
