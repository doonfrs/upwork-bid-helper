package search

import "testing"

func TestIsURL(t *testing.T) {
	for _, s := range []string{"https://www.upwork.com/x", "http://a", "file:///tmp/x.html"} {
		if !IsURL(s) {
			t.Errorf("IsURL(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"q=react", "react native", "myfeed"} {
		if IsURL(s) {
			t.Errorf("IsURL(%q) = true, want false", s)
		}
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"alias myfeed", []string{"myfeed"}, URLMyFeed},
		{"alias best", []string{"best"}, URLBestMatches},
		{"alias recent", []string{"recent"}, URLMostRecent},
		{"alias case-insensitive", []string{"Recent"}, URLMostRecent},
		{"alias saved", []string{"saved"}, URLSavedJobs},
		{"full url passthrough", []string{"https://www.upwork.com/jobs/~02abc"}, "https://www.upwork.com/jobs/~02abc"},
		{"file url passthrough", []string{"file:///tmp/x.html"}, "file:///tmp/x.html"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Resolve(c.args)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", c.args, err)
			}
			if got != c.want {
				t.Errorf("Resolve(%q) = %q, want %q", c.args, got, c.want)
			}
		})
	}

	// Search is removed: unrecognized targets (bare words, key=val, multiple
	// args) must error rather than build a search URL.
	for _, args := range [][]string{{"react"}, {"q=react native"}, {"react", "native"}, {}} {
		if got, err := Resolve(args); err == nil {
			t.Errorf("Resolve(%q) = %q, nil error; want error", args, got)
		}
	}
}
