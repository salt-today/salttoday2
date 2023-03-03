package store

const (
	// TODO: Need some lib to create batch queries for me
	InsertComment   = "INSERT INTO Comments (User, Time, Text, Likes, Dislikes) VALUES (?, ?, ?, ?, ?);"
	InsertArticle   = "INSERT INTO Article (Title, Url) VALUES (?, ?);"
	InsertUser      = "INSERT INTO Users (User) VALUES ?;"
	GetUsers        = "SELECT User FROM Users;"
	GetUserComments = "SELECT User, Time, Text, Likes, Dislikes FROM Comments WHERE User = ?"
	// TODO: Add comments by article
)
