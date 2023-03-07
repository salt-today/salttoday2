package store

import (
	"context"
	"time"
)

type Storage interface {
	AddComments(ctx context.Context, comments ...*Comment) error
	GetComments(ctx context.Context, opts CommentQueryOptions) ([]*Comment, error)
	AddArticles(ctx context.Context, articles ...*Article) error
	AddUsers(ctx context.Context, users ...*User) error
	GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error)
	SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error
	SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error
}

const (
	OrderByLiked    = iota
	OrderByDisliked = iota
	OrderByBoth     = iota
)

type CommentQueryOptions struct {
	ID          *int
	Limit       *uint
	Order       *int
	OnlyDeleted bool
	Site        *string
	UserID      *int
	UserName    *string
}

type StorageContent interface {
	[]*Comment | []*Article
}

type StoragePage[ContentType StorageContent] struct {
	Content  ContentType
	IsLast   bool
	pageData []byte
}
