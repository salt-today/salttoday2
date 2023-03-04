package store

import (
	"time"
)

type Storage interface {
	AddComments(comments ...*Comment) error
	GetUserComments(userID int) ([]*Comment, error)
	AddArticles(articles ...*Article) error
	AddUsers(users ...*User) error
	GetUnscrapedArticlesSince(scrapeThreshold time.Time) ([]*Article, error)
}
