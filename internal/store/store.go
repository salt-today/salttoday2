package store

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type storage struct {
	db *sql.DB
}

func NewStorage() (*storage, error) {
	db, err := sql.Open("mysql", "salt:salt@tcp(localhost:3306)/salt")
	if err != nil {
		return nil, err
	}

	return &storage{
		db: db,
	}, nil
}

func (s *storage) AddComments(comments []*Comment) error {
	// TODO: batch me
	for _, comment := range comments {
		_, err := s.db.Exec(InsertComment, comment.User, comment.Time, comment.Text, comment.Likes, comment.Dislikes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *storage) Shutdown() error {
	return s.db.Close()
}
