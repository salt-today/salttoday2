package rdb

// Tables
const (
	CommentsTable = "Comments"
	UsersTable    = "Users"
	ArticlesTable = "Articles"
)

// Columns
const (
	LikesSuffix    = "Likes"
	DislikesSuffix = "Dislikes"

	CommentsID        = CommentsTable + "." + "ID"
	CommentsArticleID = CommentsTable + "." + "ArticleID"
	CommentsUserID    = CommentsTable + "." + "UserID"
	CommentsTime      = CommentsTable + "." + "Time"
	CommentsText      = CommentsTable + "." + "Text"
	CommentsLikes     = CommentsTable + "." + LikesSuffix
	CommentsDislikes  = CommentsTable + "." + DislikesSuffix
	CommentsScore     = "Score"
	CommentsDeleted   = CommentsTable + "." + "Deleted"

	ArticlesID             = ArticlesTable + "." + "ID"
	ArticlesUrl            = ArticlesTable + "." + "Url"
	ArticlesTitle          = ArticlesTable + "." + "Title"
	ArticlesDiscoveryTime  = ArticlesTable + "." + "DiscoveryTime"
	ArticlesLastScrapeTime = ArticlesTable + "." + "LastScrapeTime"

	UsersID   = UsersTable + "." + "ID"
	UsersName = UsersTable + "." + "Name"

	NewAlias         = "NewAlias"
	NewAliasLikes    = NewAlias + "." + LikesSuffix
	NewAliasDislikes = NewAlias + "." + DislikesSuffix
)
