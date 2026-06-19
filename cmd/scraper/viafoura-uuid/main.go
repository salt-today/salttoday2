// Command viafoura-uuid is a one-time discovery tool. It loads a Viafoura-backed
// article in a real browser, intercepts the comment widget's request to
// livecomments.viafoura.co, and prints the section_uuid (a per-site constant) plus
// the raw comment JSON response. Use it to populate ViafouraSectionUUID in the site
// config and to confirm the comment field schema. Not part of the scraping hot path.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// https://livecomments.viafoura.co/v4/livecomments/{section}/{container}/comments?...
var livecommentsRe = regexp.MustCompile(`/v4/livecomments/([0-9a-f-]{36})/([0-9a-f-]{36})/`)

func main() {
	url := "https://lfpress.com/business/local-business/toronto-based-coffee-chain-dark-horse-opening-second-london-spot"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	pw, err := playwright.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "playwright run: %v\n", err)
		os.Exit(1)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-blink-features=AutomationControlled"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "launch: %v\n", err)
		os.Exit(1)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "new page: %v\n", err)
		os.Exit(1)
	}

	seenSection := ""
	page.On("request", func(req playwright.Request) {
		if !strings.Contains(req.URL(), "viafoura") {
			return
		}
		if m := livecommentsRe.FindStringSubmatch(req.URL()); m != nil && seenSection == "" {
			seenSection = m[1]
		}
		fmt.Printf("[req] %s %s\n", req.Method(), req.URL())
	})

	if _, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(45000),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "goto: %v\n", err)
	}

	// The comment widget lazy-mounts when scrolled into view. Scroll to the bottom
	// a few times and give the widget time to fire its API calls.
	for range 4 {
		page.Evaluate(`() => window.scrollTo(0, document.body.scrollHeight)`)
		page.WaitForTimeout(2500)
	}
	time.Sleep(3 * time.Second)

	if seenSection != "" {
		fmt.Printf("\n==> section_uuid = %s\n", seenSection)
	} else {
		fmt.Printf("\n==> no livecomments request captured (widget may not have mounted)\n")
	}
}
