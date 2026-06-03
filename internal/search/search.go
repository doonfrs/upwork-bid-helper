// Package search resolves CLI arguments to an Upwork target URL: a find-work
// feed shortcut (myfeed, best, recent, saved) or a full URL.
package search

import (
	"fmt"
	"strings"
)

const findWork = "https://www.upwork.com/nx/find-work/"

// Known Upwork page URLs, exposed as constants for reuse. These mirror the
// find-work tabs: My Feed (the base route), Best Matches, Most Recent, Saved Jobs.
const (
	// URLMyFeed is the "My Feed" tab (the default find-work page).
	URLMyFeed = findWork
	// URLBestMatches is the "Best Matches" tab.
	URLBestMatches = findWork + "best-matches"
	// URLMostRecent is the "Most Recent" tab.
	URLMostRecent = findWork + "most-recent"
	// URLSavedJobs is the "Saved Jobs" tab.
	URLSavedJobs = findWork + "saved-jobs"
)

// aliases map short names to full Upwork find-work URLs, one per tab. Keys are
// matched case-insensitively after trimming.
var aliases = map[string]string{
	"myfeed": URLMyFeed,
	"best":   URLBestMatches,
	"recent": URLMostRecent,
	"saved":  URLSavedJobs,
}

// Alias resolves a shortcut name to a full URL.
func Alias(name string) (string, bool) {
	u, ok := aliases[strings.ToLower(strings.TrimSpace(name))]
	return u, ok
}

// Resolve turns CLI args into a target URL:
//   - a single full URL (http/https/file) is used as-is
//   - a single known shortcut (myfeed, best, recent, saved) expands to its URL
//
// Anything else is unsupported (keyword search is blocked by Upwork's bot
// challenge, so it is not built here).
func Resolve(args []string) (string, error) {
	if len(args) == 1 {
		if IsURL(args[0]) {
			return args[0], nil
		}
		if u, ok := Alias(args[0]); ok {
			return u, nil
		}
	}
	return "", fmt.Errorf("unrecognized target %q — use a page (myfeed, best, recent, saved) or a full Upwork URL", strings.Join(args, " "))
}

// IsURL reports whether s looks like a full URL rather than a key=val arg.
// file:// is accepted so the tool can export from a saved HTML page offline.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "file://")
}
