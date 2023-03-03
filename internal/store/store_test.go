package store

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// This test will fuck up your local Comments table if it fails and junk. Just here to smoke test.
// Will remove, probably.
func TestNewStorage(t *testing.T) {
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
