package store

import "time"

type Comment struct {
	User     string
	Time     time.Time
	Text     string
	Likes    int
	Dislikes int
}

type Article struct {
	Title string
	Url   string
}
