package store

import (
	"context"
	"time"
)

type Storage interface {
	AddComments(ctx context.Context, comments ...*Comment) error
	GetUserComments(ctx context.Context, userID int) ([]*Comment, error)
	AddArticles(ctx context.Context, articles ...*Article) error
	AddUsers(ctx context.Context, users ...*User) error
	GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error)
}
