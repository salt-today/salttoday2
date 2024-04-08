package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/store"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Expected article url as argument")
		os.Exit(1)
	}
	articleUrl := os.Args[1]
	articleID := getArticleId(articleUrl)

	comments, _ := scraper.ScrapeCommentsFromArticles(context.Background(), []*store.Article{{ID: articleID, Url: articleUrl}})
	for _, comment := range comments {
		fmt.Printf("ID: %d\nText: %s\n\n", comment.ID, comment.Text)
	}
}

func getArticleId(url string) int {
	articleIdStr := url[strings.LastIndex(url, "-")+1:]
	articleId, err := strconv.Atoi(articleIdStr)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to convert article id from string to int %w", err))
		return 0
	}
	return articleId
}
