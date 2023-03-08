package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	stor "github.com/salt-today/salttoday2/internal/store"
)

func main() {
	store, err := stor.NewSQLStorage(context.TODO())
	if err != nil {
		panic(err)
	}

	file, err := os.Open("data/users.txt")
	if err != nil {
		panic(err)
	}

	fileScanner := bufio.NewScanner(file)

	users := make([]*stor.User, 0)
	userIDCounter := 0
	for fileScanner.Scan() {
		users = append(users, &stor.User{
			UserName: fileScanner.Text(),
			ID:       userIDCounter,
		})
		userIDCounter++
	}

	err = store.AddUsers(context.TODO(), users...)
	if err != nil {
		panic(err)
	}

	file, err = os.Open("data/articles.txt")
	if err != nil {
		panic(err)
	}
	fileScanner = bufio.NewScanner(file)
	articles := make([]*stor.Article, 0)
	articleIDCounter := 0
	for fileScanner.Scan() {
		title := fileScanner.Text()
		url := fmt.Sprintf("https://sootoday.com/%s-%d", strings.Replace(strings.ToLower(title), " ", "-", -1), articleIDCounter)

		articles = append(articles, &stor.Article{
			Url:   url,
			Title: title,
			ID:    articleIDCounter,
		})
		articleIDCounter++
	}

	err = store.AddArticles(context.TODO(), articles...)
	if err != nil {
		panic(err)
	}

	file, err = os.Open("data/comments.txt")
	if err != nil {
		panic(err)
	}
	fileScanner = bufio.NewScanner(file)

	commentsText := make([]string, 0)
	for fileScanner.Scan() {
		commentsText = append(commentsText, fileScanner.Text())
	}

	comments := make([]*stor.Comment, 2000)
	for i := 0; i < len(comments); i++ {
		deletedInt := rand.Int31() % 20 // one in 20 chance to be deleted
		deleted := false
		if deletedInt == 7 {
			deleted = true
		}

		comments[i] = &stor.Comment{
			ID:        i,
			ArticleID: i % len(articles),
			UserID:    i % len(users),
			Text:      commentsText[i%len(commentsText)],
			Time:      randomDate(),
			Likes:     rand.Int31() % 100,
			Dislikes:  rand.Int31() % 100,
			Deleted:   deleted,
		}
	}
	err = store.AddComments(context.TODO(), comments...)
	if err != nil {
		panic(err)
	}

}

func randomDate() time.Time {
	min := time.Date(2022, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Now().Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}
