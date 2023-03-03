package store

import "time"

type Comment struct {
	User     string
	Time     time.Time
	Text     string
	Likes    int32
	Dislikes int32
}

type Article struct {
	Title string
	Url   string
}
