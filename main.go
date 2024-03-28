package main

import (
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/pkg/scraper"
	"github.com/salt-today/salttoday2/pkg/server"
)

func main() {
	flag.Usage = func() {
		fmt.Println("\nSalttoday scrapes comments from various news sites and displays them.")
		fmt.Println("\nUsage:")
		fmt.Println("\n\tsalttoday <command> [arguments]")
		fmt.Println("\nCommands:")
		fmt.Println(`  server    Runs the HTTP server`)
		fmt.Println(`  scraper   Runs the scraper to collect and store comments`)
		fmt.Println()
	}
	flag.Parse()

	switch flag.Arg(0) {
	case `server`:
		server.Main()
	case `scraper`:
		scraper.Main()
	default:
		if len(flag.Arg(0)) == 0 {
			logrus.Error(`command not provided`)
		} else {
			logrus.Errorf(`%q is not a valid command.`, flag.Arg(0))
		}
		flag.Usage()
	}
}
