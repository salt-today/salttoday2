package store

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
)

// These tests will fuck up your local Comments table if it fails and junk. Just here to smoke test.
// Good to validate our db service is working, but not much else.
// Should probably also mock the DB, but meh.
// Will remove, probably.
func TestStorage_Comments(t *testing.T) {
	store, err := NewSQLStorage(context.Background())
	require.NoError(t, err)

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)

	users := []int{1, 2, 3, 4, 5}

	texts := []string{
		"I think this is cool",
		"i think this is dumb",
		"i think you're all dumb",
		"we should protest this",
		"they should all be fired",
	}

	allComments := make([]*Comment, 0, 100)
	commentsByUser := make(map[int][]*Comment)
	for i := 0; i < 1; i++ {
		user := users[i%len(users)]
		comment := &Comment{
			ID:       i,
			User:     User{ID: user},
			Time:     time.Now().Truncate(time.Second),
			Text:     texts[i%len(texts)],
			Likes:    rand.Int31(),
			Dislikes: rand.Int31(),
		}

		allComments = append(allComments, comment)

		commentsByUser[user] = append(commentsByUser[user], comment)
	}

	require.NoError(t, store.AddComments(context.Background(), allComments...))
	for _, user := range users {
		opts := CommentQueryOptions{UserID: aws.Int(user)}
		comments, err := store.GetComments(context.Background(), opts)
		require.NoError(t, err)
		require.ElementsMatch(t, commentsByUser[user], comments)
	}

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)
}

func TestStorage_Articles(t *testing.T) {
	store, err := NewSQLStorage(context.Background())
	require.NoError(t, err)

	_, err = store.db.Exec("TRUNCATE TABLE Articles;")
	require.NoError(t, err)

	titles := []string{
		"theres a lot of snow",
		"people in front of station mall protesting masks",
		"algoma U running a grad school program lul",
		"opioid crisis 2.0",
		"cool title",
	}

	urls := []string{
		"www.sootoday.com/legit/12013",
		"www.sootoday.com/legit/120qwe",
		"www.sootoday.com/legit/120fff",
		"www.sootoday.com/legit/1204324",
		"www.sootoday.com/legit/120123",
	}

	numArts := 10
	allArticles := make([]*Article, 0, numArts)
	toScrapeTimes := make([]time.Time, 0, numArts)
	artIDs := make([]int, 0, numArts)
	start := time.Now()
	toScrapeTime := time.Now()
	for i := 0; i < numArts; i++ {
		article := &Article{
			ID:             i,
			Title:          titles[i%len(titles)],
			Url:            urls[i%len(urls)],
			DiscoveryTime:  time.Now().Truncate(time.Second),
			LastScrapeTime: time.Now().Truncate(time.Second),
		}
		allArticles = append(allArticles, article)

		toScrapeTime = toScrapeTime.Add(-time.Second).Truncate(time.Second)
		toScrapeTimes = append(toScrapeTimes, toScrapeTime)
		artIDs = append(artIDs, i)
	}

	require.NoError(t, store.AddArticles(context.Background(), allArticles...))
	arts, err := store.GetUnscrapedArticlesSince(context.Background(), start)
	require.NoError(t, err)
	require.ElementsMatch(t, allArticles, arts)

	// nothing has been scraped yet, should still get all articles
	dur := time.Second * time.Duration(numArts/2+1)
	arts, err = store.GetUnscrapedArticlesSince(context.Background(), start.Add(-dur))
	require.ElementsMatch(t, allArticles, arts)

	for i := 0; i < numArts; i++ {
		require.NoError(t, store.SetArticleScrapedAt(context.Background(), toScrapeTimes[i], allArticles[i].ID))
		allArticles[i].LastScrapeTime = toScrapeTimes[i]
	}
	arts, err = store.GetUnscrapedArticlesSince(context.Background(), start)
	// should still have everything
	require.ElementsMatch(t, allArticles, arts)

	arts, err = store.GetUnscrapedArticlesSince(context.Background(), toScrapeTimes[numArts/2])
	require.ElementsMatch(t, allArticles[numArts/2:], arts)

	require.NoError(t, store.SetArticleScrapedNow(context.Background(), artIDs...))

	arts, err = store.GetUnscrapedArticlesSince(context.Background(), time.Now().Add(-time.Second))
	require.NoError(t, err)
	require.Empty(t, arts)

	_, err = store.db.Exec("TRUNCATE TABLE Articles;")
	require.NoError(t, err)
}
