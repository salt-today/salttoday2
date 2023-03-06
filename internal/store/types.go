package store

import "time"

type Comment struct {
	ID        int
	ArticleID int
	UserID    int
	Name      string
	Time      time.Time
	Text      string
	Likes     int32
	Dislikes  int32
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