package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/salt-today/salttoday2/internal/logger"
	scrpr "github.com/salt-today/salttoday2/internal/scraper"
)

func main() {
	startTime := time.Now()
	ctx := context.Background()
	log := logger.New(ctx)

	log.Info("========================================")
	log.Info("Scraper starting...")
	log.WithField("args", os.Args).Info("Command-line arguments")

	// Validate command-line arguments
	if len(os.Args) < 2 {
		log.Error("Expected days ago as an argument (e.g., ./scraper 14)")
		log.Info("Usage: scraper <days-ago> [force-scrape-bool]")
		os.Exit(1)
	}

	daysAgoArg := os.Args[1]
	daysAgo, err := strconv.Atoi(daysAgoArg)
	if err != nil {
		log.WithError(err).WithField("arg", daysAgoArg).Error("Unable to convert days ago from string to int")
		os.Exit(1)
	}

	forceScrape := false
	if len(os.Args) > 2 {
		forceStr := os.Args[2]
		forceScrape, err = strconv.ParseBool(forceStr)
		if err != nil {
			log.WithError(err).WithField("arg", forceStr).Error("Unable to convert force scrape from string to bool")
			os.Exit(1)
		}
	}

	log.WithField("days_ago", daysAgo).WithField("force_scrape", forceScrape).Info("Configuration validated")

	mysqlURL := os.Getenv("MYSQL_URL")
	if mysqlURL == "" {
		log.Error("MYSQL_URL environment variable is not set")
		os.Exit(1)
	}

	log.Info("Creating scraper instance...")
	scraper, err := scrpr.NewScraper(ctx, nil) // Use default config
	if err != nil {
		log.WithError(err).Fatal("Failed to create scraper")
	}
	defer scraper.Close()
	log.Info("Scraper instance created successfully")

	// Scrape articles
	log.Info("Starting article scraping...")
	if err := scraper.ScrapeAndStoreArticles(ctx); err != nil {
		log.WithError(err).Error("Article scraping failed (continuing to comments)")
	} else {
		log.Info("Article scraping completed successfully")
	}

	// Scrape comments
	log.Info("Starting comment scraping...")
	if err := scraper.ScrapeAndStoreComments(ctx, daysAgo, forceScrape); err != nil {
		log.WithError(err).Error("Comment scraping failed")
	} else {
		log.Info("Comment scraping completed successfully")
	}

	duration := time.Since(startTime)
	log.WithField("duration", duration).Info("========================================")
	log.WithField("duration", duration).Info("Scraping complete - exiting normally")
	log.Info("========================================")
}
