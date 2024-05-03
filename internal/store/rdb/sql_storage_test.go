package rdb

import (
	"context"
	"testing"
	"time"

	"github.com/salt-today/salttoday2/internal/store"
	"github.com/stretchr/testify/require"
)

func TestAddArticle(t *testing.T) {
	s, err := New(context.Background())
	require.NoError(t, err)

	article1 := &store.Article{
		ID:            1,
		Title:         "Article 1",
		Url:           "testurl1",
		SiteName:      "all",
		DiscoveryTime: time.Now(),
	}
	article2 := &store.Article{
		ID:            1,
		Title:         "Article 2",
		Url:           "testurl1",
		SiteName:      "site2",
		DiscoveryTime: time.Now(),
	}

	articles := []*store.Article{article1, article2}

	err = s.AddArticles(context.Background(), articles...)
	require.NoError(t, err)
}
