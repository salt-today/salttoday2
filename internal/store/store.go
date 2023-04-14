package store

import (
	"context"
	"time"
)

type Storage interface {
	AddComments(ctx context.Context, comments ...*Comment) error
	GetComments(ctx context.Context, opts CommentQueryOptions) ([]*Comment, error)
	AddArticles(ctx context.Context, articles ...*Article) error
	GetArticles(ctx context.Context, articleIDs ...int) ([]*Article, error)
	AddUsers(ctx context.Context, users ...*User) error
	GetUsersByIDs(ctx context.Context, userIDs ...int) ([]*User, error)
	GetUserByName(ctx context.Context, userName string) (*User, error)
	QueryUsers(ctx context.Context, opts UserQueryOptions) ([]*User, error)
	GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error)
	SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error
	SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error
}

const (
	OrderByLiked    = iota
	OrderByDisliked = iota
	OrderByBoth     = iota
)

type PageQueryOptions struct {
	Limit *uint
	Page  *uint
	Order *int
}

type CommentQueryOptions struct {
	ID          *int
	UserID      *int
	OnlyDeleted bool
	Site        *string
	DaysAgo     *uint
	Format      *string

	PageOpts PageQueryOptions
}

type UserQueryOptions struct {
	ID   *int
	Site *string

	PageOpts PageQueryOptions
}

type StorageContent interface {
	[]*Comment | []*Article
}

type StoragePage[ContentType StorageContent] struct {
	Content  ContentType
	IsLast   bool
	pageData []byte
}
