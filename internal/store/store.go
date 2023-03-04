package store

import (
	"context"
	"time"
)

type Storage interface {
	AddComments(ctx context.Context, comments ...*Comment) error
	GetUserComments(ctx context.Context, userID int, opts QueryOptions) ([]*Comment, error)
	AddArticles(ctx context.Context, articles ...*Article) error
	AddUsers(ctx context.Context, users ...*User) error
	GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error)
	SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error
	SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error
}

const (
	OrderByLiked    = iota
	OrderByDisliked = iota
)

type QueryOptions struct {
	Limit       *uint
	Order       *int
	ShowDeleted *bool
	City        *bool
}

type StorageContent interface {
	[]*Comment | []*Article
}

type StoragePage[ContentType StorageContent] struct {
	Content  ContentType
	IsLast   bool
	pageData []byte
}
