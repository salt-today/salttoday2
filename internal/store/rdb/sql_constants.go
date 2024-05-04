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
	DeletedSuffix  = "Deleted"
	SiteNameSuffix = "SiteName"

	CommentsID        = CommentsTable + "." + "ID"
	CommentsArticleID = CommentsTable + "." + "ArticleID"
	CommentsUserID    = CommentsTable + "." + "UserID"
	CommentsTime      = CommentsTable + "." + "Time"
	CommentsText      = CommentsTable + "." + "Text"
	CommentsLikes     = CommentsTable + "." + LikesSuffix
	CommentsDislikes  = CommentsTable + "." + DislikesSuffix
	CommentsScore     = "Score"
	CommentsDeleted   = CommentsTable + "." + DeletedSuffix

	CommentControverstyView            = "CommentControversy"
	CommentControverstyID              = CommentControverstyView + "." + "ID"
	CommentControverstyWeightedEntropy = CommentControverstyView + "." + "WeightedEntropy"

	ArticlesID             = ArticlesTable + "." + "ID"
	ArticlesSiteName       = ArticlesTable + "." + SiteNameSuffix
	ArticlesUrl            = ArticlesTable + "." + "Url"
	ArticlesTitle          = ArticlesTable + "." + "Title"
	ArticlesDiscoveryTime  = ArticlesTable + "." + "DiscoveryTime"
	ArticlesLastScrapeTime = ArticlesTable + "." + "LastScrapeTime"

	UsersID      = UsersTable + "." + "ID"
	UsersName    = UsersTable + "." + "Name"
	UserLikes    = "UserLikes"
	UserDislikes = "UserDislikes"
	UserScore    = "UserScore"

	SiteLikes    = "SiteLikes"
	SiteDislikes = "SiteDislikes"
	SiteScore    = "SiteScore"

	NewAlias         = "NewAlias"
	NewAliasLikes    = NewAlias + "." + LikesSuffix
	NewAliasDislikes = NewAlias + "." + DislikesSuffix
	NewAliasDeleted  = NewAlias + "." + DeletedSuffix

	NewAliasSiteName = NewAlias + "." + SiteNameSuffix

	OldAlias = "OldAlias"
)
