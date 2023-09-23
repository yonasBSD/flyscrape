// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/inancgumus/screen"
	"github.com/philippta/flyscrape"
)

type DevCommand struct{}

func (c *DevCommand) Run(args []string) error {
	fs := flag.NewFlagSet("flyscrape-dev", flag.ContinueOnError)
	proxy := fs.String("proxy", "", "proxy")
	fs.Usage = c.Usage

	if err := fs.Parse(args); err != nil {
		return err
	} else if fs.NArg() == 0 || fs.Arg(0) == "" {
		return fmt.Errorf("script path required")
	} else if fs.NArg() > 1 {
		return fmt.Errorf("too many arguments")
	}

	var fetch flyscrape.FetchFunc
	if *proxy != "" {
		fetch = flyscrape.ProxiedFetch(*proxy)
	} else {
		fetch = flyscrape.Fetch()
	}

	fetch = flyscrape.CachedFetch(fetch)
	script := fs.Arg(0)

	err := flyscrape.Watch(script, func(s string) error {
		opts, scrape, err := flyscrape.Compile(s)
		if err != nil {
			screen.Clear()
			screen.MoveTopLeft()

			if errs, ok := err.(interface{ Unwrap() []error }); ok {
				for _, err := range errs.Unwrap() {
					log.Printf("%s:%v\n", script, err)
				}
			} else {
				log.Println(err)
			}

			// ignore compilation errors
			return nil
		}

		scraper := flyscrape.NewScraper()
		scraper.ScrapeFunc = scrape
		flyscrape.LoadModules(scraper, opts)

		scraper.Run()

		scraper.OnResponse(func(resp *flyscrape.Response) {
			screen.Clear()
			screen.MoveTopLeft()
			if resp.Error != nil {
				log.Println(resp.Error)
				return
			}
			fmt.Println(flyscrape.PrettyPrint(resp.ScrapeResult, ""))
		})

		return nil
	})
	if err != nil && err != flyscrape.StopWatch {
		return fmt.Errorf("failed to watch script %q: %w", script, err)
	}

	return nil
}

func (c *DevCommand) Usage() {
	fmt.Println(`
The dev command watches the scraping script and re-runs it on any change.
Recursive scraping is disabled in this mode, only the initial URL will be scraped.

Usage:

    flyscrape dev SCRIPT


Examples:

    # Run and watch script.
    $ flyscrape dev example.js
`[1:])
}