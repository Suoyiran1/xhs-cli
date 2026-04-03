package cmd

import (
	"github.com/Suoyiran1/xhs-cli/internal/browser"
	"github.com/Suoyiran1/xhs-cli/internal/configs"
	"github.com/go-rod/rod"
)

// newBrowser creates and returns a headless browser instance.
func newBrowserInstance(headless bool) *browser.BrowserWrapper {
	var opts []browser.Option
	if binPath := configs.GetBinPath(); binPath != "" {
		opts = append(opts, browser.WithBinPath(binPath))
	}
	return &browser.BrowserWrapper{
		Browser: browser.NewBrowser(headless, opts...),
	}
}

// withPage creates a browser, opens a page, runs fn, then cleans up.
func withPage(fn func(page *rod.Page) error) error {
	b := newBrowserInstance(configs.IsHeadless())
	defer b.Browser.Close()

	page := b.Browser.NewPage()
	return fn(page)
}

// withPageNoHeadless creates a browser with headless=false (for login QR code).
func withPageNoHeadless(fn func(page *rod.Page) error) error {
	b := newBrowserInstance(false)
	defer b.Browser.Close()

	page := b.Browser.NewPage()
	return fn(page)
}
