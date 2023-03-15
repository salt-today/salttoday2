package store

import (
	"fmt"
	"time"
)

type Comment struct {
	ID        int
	ArticleID int
	UserID    int
	Time      time.Time
	Text      string
	Likes     int32
	Dislikes  int32
	Deleted   bool
}

type Article struct {
	ID             int
	Title          string
	Url            string
	DiscoveryTime  time.Time
	LastScrapeTime *time.Time
}

type User struct {
	ID       int
	UserName string
}

type NoQueryResultsError struct{}

func (e NoQueryResultsError) Error() string {
	return fmt.Sprintf("No results found")
}
