// Package search builds an Upwork job-search URL from key=value arguments.
package search

import (
	"net/url"
	"strings"
)

// BaseURL is the Upwork job search endpoint.
const BaseURL = "https://www.upwork.com/nx/search/jobs/"

// IsURL reports whether s looks like a full URL rather than a key=val arg.
// file:// is accepted so the tool can export from a saved HTML page offline.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "file://")
}

// ParseArgs turns ["q=react native", "category=web"] into a map. Values may
// contain '='; only the first '=' splits. Args without '=' become a bare query
// term appended to q.
func ParseArgs(args []string) map[string]string {
	m := map[string]string{}
	var terms []string
	for _, a := range args {
		if i := strings.Index(a, "="); i >= 0 {
			k := strings.TrimSpace(a[:i])
			v := a[i+1:]
			if k != "" {
				m[k] = v
			}
		} else if a != "" {
			terms = append(terms, a)
		}
	}
	if len(terms) > 0 {
		joined := strings.Join(terms, " ")
		if existing, ok := m["q"]; ok && existing != "" {
			m["q"] = existing + " " + joined
		} else {
			m["q"] = joined
		}
	}
	return m
}

// BuildURL assembles the search URL from parsed args. Unknown keys are passed
// through as query parameters so the tool stays useful as Upwork adds filters.
func BuildURL(args map[string]string) string {
	q := url.Values{}
	for k, v := range args {
		q.Set(k, v)
	}
	if len(q) == 0 {
		return BaseURL
	}
	return BaseURL + "?" + q.Encode()
}
