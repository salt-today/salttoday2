package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/salt-today/salttoday2/internal/scraper"
	"github.com/salt-today/salttoday2/internal/sdk"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Expected days ago as an argument")
		os.Exit(1)
	}
	daysAgoArg := os.Args[1]
	daysAgo, err := strconv.Atoi(daysAgoArg)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to convert days ago from string to int %w", err))
		os.Exit(1)
	}

	ctx := context.Background()

	scraper.ScrapeAndStoreArticles(ctx)
	scraper.ScrapeAndStoreComments(ctx, daysAgo)

	sdk.Logger(ctx).Info("Scraping complete")
}
