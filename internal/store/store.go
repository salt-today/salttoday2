package store

import (
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Storage interface {
	AddComments(comments ...*Comment) error
	GetUserComments(userID int) ([]*Comment, error)
	AddArticles(articles ...*Article) error
	AddUsers(users ...*User) error
	GetUnscrapedArticlesSince(scrapeThreshold time.Time) ([]*Article, error)
	SetArticleScrapedNow(articleIDs ...int) error
	SetArticleScrapedAt(scrapedTime time.Time, articleIDs ...int)
}
