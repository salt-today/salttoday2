package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/salt-today/salttoday2/internal/logger"
	scrpr "github.com/salt-today/salttoday2/internal/scraper"
)

func main() {
	logger := logger.New(context.Background())
	if len(os.Args) < 2 {
		fmt.Println("Expected days ago as an argument")
		os.Exit(1)
	}
	daysAgoArg := os.Args[1]
	daysAgo, err := strconv.Atoi(daysAgoArg)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to convert days ago from string to int %w", err))
		os.Exit(1)
	}

	forceScrape := false
	if len(os.Args) > 2 {
		forceStr := os.Args[2]
		forceScrape, err = strconv.ParseBool(forceStr)
		if err != nil {
			fmt.Println(fmt.Errorf("unable to convert force scrape from string to bool %w", err))
			os.Exit(1)
		}
	}

	ctx := context.Background()

	// Use the main scraper
	scraper, err := scrpr.NewScraper(ctx, nil) // Use default config
	if err != nil {
		logger.WithError(err).Fatal("Failed to create scraper")
	}
	defer scraper.Close()

	// Scrape articles
	if err := scraper.ScrapeAndStoreArticles(ctx); err != nil {
		logger.WithError(err).Error("Article scraping failed")
	}

	// Scrape comments
	if err := scraper.ScrapeAndStoreComments(ctx, daysAgo, forceScrape); err != nil {
		logger.WithError(err).Error("Comment scraping failed")
	}

	logger.Info("Scraping complete")
}
