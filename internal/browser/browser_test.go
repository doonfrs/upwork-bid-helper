package browser

import (
	"path/filepath"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

// TestCookieStateRoundTrip proves a session cookie (no expiry — the kind Chrome
// drops on close) is saved and restored across two separate launches with
// different profiles. This is the mechanism that keeps the Upwork login alive.
func TestCookieStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")

	b1, err := Launch(Options{Headless: true, ProfileDir: filepath.Join(dir, "p1"), StateFile: state})
	if err != nil {
		t.Skipf("cannot launch chrome: %v", err)
	}
	// A session cookie: no Expires set.
	if err := b1.rod.SetCookies([]*proto.NetworkCookieParam{{
		Name: "ubh_test", Value: "42", Domain: "example.com", Path: "/",
	}}); err != nil {
		b1.Close()
		t.Fatalf("set cookie: %v", err)
	}
	if err := b1.SaveState(); err != nil {
		b1.Close()
		t.Fatalf("save state: %v", err)
	}
	b1.Close()

	// Fresh profile + same state file => the cookie must come back.
	b2, err := Launch(Options{Headless: true, ProfileDir: filepath.Join(dir, "p2"), StateFile: state})
	if err != nil {
		t.Skipf("cannot relaunch chrome: %v", err)
	}
	defer b2.Close()

	cs, err := b2.rod.GetCookies()
	if err != nil {
		t.Fatalf("get cookies: %v", err)
	}
	for _, c := range cs {
		if c.Name == "ubh_test" && c.Value == "42" {
			return // restored
		}
	}
	t.Fatalf("session cookie not restored across runs (got %d cookies)", len(cs))
}
