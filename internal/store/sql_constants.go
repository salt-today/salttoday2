package store

// Tables
const (
	CommentsTable = "Comments"
	UsersTable    = "Users"
	ArticlesTable = "Articles"
)

// Columns
const (
	CommentsID        = CommentsTable + "." + "ID"
	CommentsArticleID = CommentsTable + "." + "ArticleID"
	CommentsUserID    = CommentsTable + "." + "UserID"
	CommentsTime      = CommentsTable + "." + "Time"
	CommentsText      = CommentsTable + "." + "Text"
	CommentsLikes     = CommentsTable + "." + "Likes"
	CommentsDislikes  = CommentsTable + "." + "Dislikes"
	CommentsScore     = "Score"
	CommentsDeleted   = CommentsTable + "." + "Deleted"

	ArticlesID             = ArticlesTable + "." + "ID"
	ArticlesUrl            = ArticlesTable + "." + "Url"
	ArticlesTitle          = ArticlesTable + "." + "Title"
	ArticlesDiscoveryTime  = ArticlesTable + "." + "DiscoveryTime"
	ArticlesLastScrapeTime = ArticlesTable + "." + "LastScrapeTime"

	UsersID   = UsersTable + "." + "ID"
	UsersName = UsersTable + "." + "Name"
)
