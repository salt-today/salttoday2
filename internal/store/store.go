package store

import (
	"context"
	"time"
)

type Storage interface {
	AddComments(ctx context.Context, comments []*Comment) error
	GetComments(ctx context.Context, opts *CommentQueryOptions) ([]*Comment, error)
	AddArticles(ctx context.Context, articles ...*Article) error
	GetArticles(ctx context.Context, articleIDs ...int) ([]*Article, error)
	AddUsers(ctx context.Context, users ...*User) error
	GetUsers(ctx context.Context, opts *UserQueryOptions) ([]*User, error)
	GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error)
	SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error
	SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error
}

const (
	OrderByLikes    = iota
	OrderByDislikes = iota
	OrderByBoth     = iota
)

type PageQueryOptions struct {
	Limit *uint
	Page  *uint
	Order *int
	Site  *string
}

type CommentQueryOptions struct {
	ID          *int
	UserID      *int
	OnlyDeleted bool
	DaysAgo     *uint

	ArticleID *int

	PageOpts *PageQueryOptions
}

type UserQueryOptions struct {
	ID *int

	PageOpts PageQueryOptions
}

type StorageContent interface {
	[]*Comment | []*Article
}
