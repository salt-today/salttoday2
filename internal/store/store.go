package store

import (
	"context"
	"time"
)

type Storage interface {
	AddComments(ctx context.Context, comments []*Comment) error
	GetComments(ctx context.Context, opts *CommentQueryOptions) ([]*Comment, error)
	AddArticles(ctx context.Context, articles ...*Article) error
	GetArticles(ctx context.Context, articleIDs ...string) ([]*Article, error)
	GetRecentlyDiscoveredArticles(ctx context.Context, threshold time.Time) ([]*Article, error)
	AddUsers(ctx context.Context, users ...*User) error
	GetUsers(ctx context.Context, opts *UserQueryOptions) ([]*User, error)
	GetSites(ctx context.Context, opts *PageQueryOptions) ([]*Site, error)
	GetTopSite(ctx context.Context, orderBy int) (*Site, error)
	GetStats(ctx context.Context) (*Stats, error)
	SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...string) error
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
	ID          *string
	UserID      *string
	OnlyDeleted bool
	DaysAgo     uint
	Text        string

	ArticleID *string

	PageOpts *PageQueryOptions
}

type UserQueryOptions struct {
	ID   *string
	Name string

	PageOpts *PageQueryOptions
}
