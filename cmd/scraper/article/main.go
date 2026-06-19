package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/salt-today/salttoday2/internal/logger"
	scrpr "github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/store"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Expected article url as argument")
		os.Exit(1)
	}
	articleUrl := os.Args[1]
	// This debug tool targets Village Media URLs, whose article ID is the trailing
	// integer (e.g. .../some-title-12345).
	articleID := articleUrl[strings.LastIndex(articleUrl, "-")+1:]

	ctx := context.Background()
	logEntry := logger.New(ctx)

	scraper, err := scrpr.NewScraper(ctx, nil) // Use default config
	if err != nil {
		logEntry.WithError(err).Fatal("Failed to create scraper")
	}
	defer scraper.Close()

	comments, _, err := scraper.ScrapeCommentsFromArticle(ctx, &store.Article{ID: articleID, Url: articleUrl})
	if err != nil {
		logEntry.WithError(err).Fatal("Failed to scrape comments")
	}

	logEntry.WithField("commentsFound", len(comments)).Info("Scraping completed")

	for _, comment := range comments {
		fmt.Printf("ID: %s\nText: %s\n\n", comment.ID, comment.Text)
	}
}
