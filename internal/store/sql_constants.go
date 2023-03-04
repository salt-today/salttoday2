package store

// Tables
const (
	CommentsTable = "Comments"
	UsersTable    = "Users"
	ArticlesTable = "Articles"
)

// Columns
const (
	CommentsID       = "ID"
	CommentsUserID   = "UserID"
	CommentsTime     = "Time"
	CommentsText     = "Text"
	CommentsLikes    = "Likes"
	CommentsDislikes = "Dislikes"

	ArticlesID             = "ID"
	ArticlesUrl            = "Url"
	ArticlesTitle          = "Title"
	ArticlesDiscoveryTime  = "DiscoveryTime"
	ArticlesLastScrapeTime = "LastScrapeTime"

	UsersID   = "ID"
	UsersName = "User"
)
