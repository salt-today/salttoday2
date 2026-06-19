package store

import (
	"fmt"
	"time"
)

type Comment struct {
	ID       string
	Article  Article
	User     User
	Time     time.Time
	Text     string
	Likes    int32
	Dislikes int32
	Deleted  bool
}

type Article struct {
	ID             string
	Title          string
	SiteName       string
	Url            string
	DiscoveryTime  time.Time
	LastScrapeTime time.Time
}

type User struct {
	ID            string
	UserName      string
	TotalLikes    int32
	TotalDislikes int32
	TotalScore    int32
}

type Site struct {
	Name          string
	TotalLikes    int32
	TotalDislikes int32
	TotalScore    int32
}

type Stats struct {
	CommentCount int
	DeletedCount int
	LikeCount    int
	DislikeCount int
	ArticleCount int
	UserCount    int
}

type NoQueryResultsError struct{}

func (e *NoQueryResultsError) Error() string {
	return fmt.Sprintf("No results found")
}
