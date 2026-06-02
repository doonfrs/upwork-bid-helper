// Command upwork-bid-helper drives a real Chrome to export Upwork feed/search
// results or a single job to JSON / CSV / XML.
//
// Step 1 (this build): CLI mode only.
//
//	upwork-bid-helper [flags] <url>
//	upwork-bid-helper [flags] q="react native" category=...
//
// The browser opens visibly; log in once and the persistent profile reuses the
// session on later runs. GUI mode (double-click) arrives in a later step.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"

	"github.com/doonfrs/upwork-bid-helper/internal/browser"
	"github.com/doonfrs/upwork-bid-helper/internal/export"
	"github.com/doonfrs/upwork-bid-helper/internal/extract"
	"github.com/doonfrs/upwork-bid-helper/internal/model"
	"github.com/doonfrs/upwork-bid-helper/internal/search"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		format     = flag.String("format", "all", "output format: json | csv | xml | all")
		out        = flag.String("out", "", "output file (or prefix when --format=all); default ./upwork-<type>-<ts>")
		chromePath = flag.String("chrome", "", "path to Chrome binary (default: system Chrome)")
		profile    = flag.String("profile", "", "persistent profile dir (default: app config dir)")
		timeout    = flag.Duration("timeout", 3*time.Minute, "max wait for the page to be ready (incl. manual login)")
		headless   = flag.Bool("headless", false, "run without a window — for local file:// exports/testing only; do NOT use against live Upwork (instantly bot-flagged)")
	)
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		return fmt.Errorf("GUI mode is not implemented yet.\nUsage:\n  upwork-bid-helper <upwork-url>\n  upwork-bid-helper q=\"react native\" category=...")
	}

	target := resolveTarget(args)
	fmt.Fprintf(os.Stderr, "target: %s\n", target)

	b, err := browser.Launch(browser.Options{ProfileDir: *profile, ChromePath: *chromePath, Headless: *headless})
	if err != nil {
		return err
	}
	defer b.Close()

	page, err := b.NewPage()
	if err != nil {
		return err
	}
	if err := page.Navigate(target); err != nil {
		return fmt.Errorf("navigate: %w", err)
	}

	res, err := waitAndExtract(page, target, *timeout)
	if err != nil {
		return err
	}
	if !res.Exportable() {
		return fmt.Errorf("nothing to export (page type %q, %d jobs) — open a feed, search, or job page", res.PageType, res.Count)
	}

	formats, err := parseFormats(*format)
	if err != nil {
		return err
	}
	written, err := writeOutputs(res, formats, *out)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "exported %d job(s) from %s page:\n", count(res), res.PageType)
	for _, p := range written {
		fmt.Println(p)
	}
	return nil
}

// resolveTarget returns the URL to visit: a full URL passed directly, or one
// built from key=val search args.
func resolveTarget(args []string) string {
	if len(args) == 1 && search.IsURL(args[0]) {
		return args[0]
	}
	return search.BuildURL(search.ParseArgs(args))
}

// waitAndExtract polls until the page is ready, surfacing login/CAPTCHA to the
// user (the visible window lets them solve it), then runs the extractor.
func waitAndExtract(page *rod.Page, target string, timeout time.Duration) (*model.Result, error) {
	deadline := time.Now().Add(timeout)
	var notedChallenge bool
	renavigated := false

	for time.Now().Before(deadline) {
		switch browser.Probe(page) {
		case browser.StatusLogin, browser.StatusCaptcha:
			if !notedChallenge {
				fmt.Fprintln(os.Stderr, "→ Upwork is asking you to log in / solve a challenge in the browser window. Waiting for you to finish…")
				notedChallenge = true
			}
		case browser.StatusReady:
			// If we had been bounced to login, return to the target once.
			if notedChallenge && !renavigated && !sameURL(page, target) {
				_ = page.Navigate(target)
				renavigated = true
				time.Sleep(1500 * time.Millisecond)
				continue
			}
			res, err := extract.Run(page)
			if err != nil {
				return nil, err
			}
			if res.Exportable() || res.PageType == model.PageUnknown {
				return res, nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	// Last attempt so the user gets a result/error rather than a bare timeout.
	if res, err := extract.Run(page); err == nil {
		return res, nil
	}
	return nil, fmt.Errorf("timed out after %s waiting for the page to be ready", timeout)
}

func sameURL(page *rod.Page, target string) bool {
	info, err := page.Info()
	if err != nil {
		return false
	}
	return strings.Contains(info.URL, strings.TrimPrefix(strings.TrimPrefix(target, "https://"), "http://"))
}

func count(res *model.Result) int {
	if len(res.Jobs) > 0 {
		return len(res.Jobs)
	}
	if res.Job != nil {
		return 1
	}
	return 0
}

func parseFormats(s string) ([]export.Format, error) {
	if strings.EqualFold(s, "all") {
		return []export.Format{export.JSON, export.CSV, export.XML}, nil
	}
	var fs []export.Format
	for _, part := range strings.Split(s, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "json":
			fs = append(fs, export.JSON)
		case "csv":
			fs = append(fs, export.CSV)
		case "xml":
			fs = append(fs, export.XML)
		default:
			return nil, fmt.Errorf("unknown format %q (use json, csv, xml, or all)", part)
		}
	}
	if len(fs) == 0 {
		return nil, fmt.Errorf("no output format selected")
	}
	return fs, nil
}

// writeOutputs writes res in each format and returns the file paths written.
func writeOutputs(res *model.Result, formats []export.Format, out string) ([]string, error) {
	prefix := out
	if prefix == "" {
		prefix = fmt.Sprintf("upwork-%s-%s", res.PageType, time.Now().Format("20060102-150405"))
	}
	// If a single format and an explicit filename with extension, honor it as-is.
	explicitFile := out != "" && len(formats) == 1 && filepath.Ext(out) != ""

	var paths []string
	for _, f := range formats {
		path := prefix + "." + f.Ext()
		if explicitFile {
			path = out
		}
		file, err := os.Create(path)
		if err != nil {
			return paths, fmt.Errorf("create %s: %w", path, err)
		}
		if err := export.Write(file, res, f); err != nil {
			file.Close()
			return paths, fmt.Errorf("write %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
