package store

import (
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
	store, err := NewStorage()
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

	require.NoError(t, store.Shutdown())
}

func TestStorage_Comments(t *testing.T) {
	store, err := NewStorage()
	require.NoError(t, err)

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)

	users := []string{"userA", "userB", "userC", "userD", "userE"}
	texts := []string{
		"I think this is cool",
		"i think this is dumb",
		"i think you're all dumb",
		"we should protest this",
		"they should all be fired",
	}

	allComments := make([]*Comment, 0, 100)
	commentsByUser := make(map[string][]*Comment)
	for i := 0; i < 100; i++ {
		user := users[i%len(users)]
		comment := &Comment{
			User:     user,
			Time:     time.Now().Truncate(time.Second),
			Text:     texts[i%len(texts)],
			Likes:    rand.Int31(),
			Dislikes: rand.Int31(),
		}

		allComments = append(allComments, comment)

		commentsByUser[user] = append(commentsByUser[user], comment)
	}

	require.NoError(t, store.AddComments(allComments...))
	for _, user := range users {
		comments, err := store.GetUserComments(user)
		require.NoError(t, err)
		require.ElementsMatch(t, commentsByUser[user], comments)
	}

	_, err = store.db.Exec("TRUNCATE TABLE Comments;")
	require.NoError(t, err)
}
