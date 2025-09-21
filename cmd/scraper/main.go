package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/scraper"
)

func main() {
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

	// Use Playwright scraper for both articles and comments
	scraper.ScrapeAndStoreArticlesWithPlaywright(ctx)
	scraper.ScrapeAndStoreCommentsWithPlaywright(ctx, daysAgo, forceScrape)

	logger.New(ctx).Info("Scraping complete")
}
