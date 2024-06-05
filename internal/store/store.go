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
	GetSites(ctx context.Context, opts *PageQueryOptions) ([]*Site, error)
	GetTopSite(ctx context.Context, orderBy int) (*Site, error)
	GetStats(ctx context.Context) (*Stats, error)
	SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error
}

const (
	OrderByLikes         = iota
	OrderByDislikes      = iota
	OrderByBoth          = iota
	OrderByControversial = iota
)

type PageQueryOptions struct {
	Limit *uint
	Page  *uint
	Order int
	Site  string
}

type CommentQueryOptions struct {
	ID          *int
	UserID      *int
	OnlyDeleted bool
	DaysAgo     uint
	Text        string

	ArticleID *int

	PageOpts *PageQueryOptions
}

type UserQueryOptions struct {
	ID   *int
	Name string

	PageOpts *PageQueryOptions
}
