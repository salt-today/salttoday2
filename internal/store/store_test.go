package store

import (
	"context"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

// These tests will fuck up your local Comments table if it fails and junk. Just here to smoke test.
// Good to validate our db service is working, but not much else.
// Should probably also mock the DB, but meh.
// Will remove, probably.
func TestBasicDB(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	store, err := NewSQLStorage(testCtx)
	require.NoError(t, err)

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)

	rows, err := store.db.Query("SELECT * FROM Comments;")
	require.NoError(t, err)
	require.False(t, rows.Next())
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	expectedUser := "someUser"
	expectedComment := "someComment"

	result, err := store.db.Exec("INSERT INTO Comments (User, Text) VALUES (?, ?)", "someUser", "someComment")
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.Equal(t, rowsAffected, int64(1))

	rows, err = store.db.Query("SELECT User, Text FROM Comments;")
	require.NoError(t, err)
	require.True(t, rows.Next())
	var user, comment string
	require.NoError(t, rows.Scan(&user, &comment))
	require.Equal(t, expectedUser, user)
	require.Equal(t, expectedComment, comment)
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)

	require.NoError(t, store.shutdown())
}

func TestStorage_Comments(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	store, err := NewSQLStorage(testCtx)
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
			UserID:   user,
			Time:     time.Now().Truncate(time.Second),
			Text:     texts[i%len(texts)],
			Likes:    rand.Int31(),
			Dislikes: rand.Int31(),
		}

		allComments = append(allComments, comment)

		commentsByUser[user] = append(commentsByUser[user], comment)
	}

	require.NoError(t, store.AddComments(testCtx, allComments...))
	for _, user := range users {
		comments, err := store.GetUserComments(testCtx, user)
		require.NoError(t, err)
		require.ElementsMatch(t, commentsByUser[user], comments)
	}

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)
}

func TestStorage_Articles(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	store, err := NewSQLStorage(testCtx)
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
	//start := time.Now()
	now := time.Now()
	for i := 0; i < numArts; i++ {
		now = now.Add(-time.Second)
		article := &Article{
			ID:             i,
			Title:          titles[i%len(titles)],
			Url:            urls[i%len(urls)],
			DiscoveryTime:  now.Truncate(time.Second),
			LastScrapeTime: nil,
		}

		allArticles = append(allArticles, article)
	}

	require.NoError(t, store.AddArticles(testCtx, allArticles...))
	arts, err := store.GetUnscrapedArticlesSince(testCtx, time.Now())
	require.ElementsMatch(t, allArticles, arts)

	dur := time.Second * time.Duration(numArts/2)
	arts, err = store.GetUnscrapedArticlesSince(testCtx, time.Now().Add(-dur))
	require.ElementsMatch(t, allArticles[:5], arts[:5])

	_, err = store.db.Exec("TRUNCATE TABLE Articles;")
	require.NoError(t, err)
}
