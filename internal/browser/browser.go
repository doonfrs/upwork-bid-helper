// Package browser launches and controls a real Chrome via go-rod, using a
// persistent profile so the logged-in Upwork session is reused across runs.
package browser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Status is the high-level state of a loaded page, used to decide whether the
// window must be surfaced for the human (login / CAPTCHA) or is ready to scrape.
type Status string

const (
	StatusReady   Status = "ready"   // window.__NUXT__ is present; safe to extract
	StatusLogin   Status = "login"   // redirected to a login page
	StatusCaptcha Status = "captcha" // Cloudflare / PerimeterX challenge visible
	StatusLoading Status = "loading" // not ready yet
)

// Options configures a browser launch.
type Options struct {
	ProfileDir string // persistent user-data-dir; defaults to the app config dir
	ChromePath string // explicit Chrome binary; defaults to the system Chrome
	// Headless launches without a window. ONLY for local/offline file:// tests —
	// never against live Upwork, where headless is instantly bot-flagged.
	Headless bool
}

// Browser wraps a launched Chrome and its launcher for clean teardown.
type Browser struct {
	launcher   *launcher.Launcher
	rod        *rod.Browser
	profileDir string
}

// DefaultProfileDir returns the app-owned persistent profile directory.
func DefaultProfileDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, "upwork-bid-helper", "profile")
}

// Launch starts Chrome and connects to it.
func Launch(opts Options) (*Browser, error) {
	profile := opts.ProfileDir
	if profile == "" {
		profile = DefaultProfileDir()
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		return nil, fmt.Errorf("create profile dir: %w", err)
	}

	l := launcher.New().
		UserDataDir(profile).
		Headless(opts.Headless).
		Leakless(false). // avoid the AV-flagged helper binary; we close Chrome ourselves
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox")

	if bin := opts.ChromePath; bin != "" {
		l = l.Bin(bin)
	} else if path, ok := launcher.LookPath(); ok {
		l = l.Bin(path) // prefer the user's real Chrome over a managed download
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chrome: %w (is Chrome installed? profile in use by another window?)", err)
	}

	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		l.Kill()
		return nil, fmt.Errorf("connect to chrome: %w", err)
	}
	return &Browser{launcher: l, rod: b, profileDir: profile}, nil
}

// NewPage opens a blank page.
func (b *Browser) NewPage() (*rod.Page, error) {
	return b.rod.Page(proto.TargetCreateTarget{})
}

// Close shuts the browser down and reaps the Chrome process.
func (b *Browser) Close() {
	if b.rod != nil {
		_ = b.rod.Close()
	}
	if b.launcher != nil {
		b.launcher.Kill()
	}
}

// statusJS classifies the page state from within the page.
const statusJS = `() => {
  const p = location.pathname;
  if (/\/ab\/account-security\/login|\/login\b/.test(p)) return 'login';
  if (document.querySelector('.cf-turnstile, iframe[src*="challenges.cloudflare.com"], [data-sitekey]')) return 'captcha';
  if (document.querySelector('#px-captcha, [id^="px-captcha"], iframe[src*="captcha"]')) return 'captcha';
  const body = document.body ? document.body.innerText.slice(0, 2000) : '';
  if (/press\s*&\s*hold/i.test(body)) return 'captcha';
  if (window.__NUXT__) return 'ready';
  return 'loading';
}`

// Probe classifies the current page state (login / captcha / ready / loading).
func Probe(page *rod.Page) Status {
	obj, err := page.Eval(statusJS)
	if err != nil {
		return StatusLoading
	}
	switch Status(obj.Value.Str()) {
	case StatusLogin:
		return StatusLogin
	case StatusCaptcha:
		return StatusCaptcha
	case StatusReady:
		return StatusReady
	default:
		return StatusLoading
	}
}

// Auth is the authentication state observed on an auth-gated Upwork page.
type Auth string

const (
	AuthIn      Auth = "in"      // signed in (an /nx/ app route rendered)
	AuthLogin   Auth = "login"   // on a login / signup / account-security page
	AuthCaptcha Auth = "captcha" // a challenge is blocking
	AuthUnknown Auth = "unknown" // still loading / indeterminate
)

// authJS runs on an auth-gated route. Logged-out users get bounced to a login
// page, so reaching an /nx/ app route with the SPA initialized means signed in.
const authJS = `() => {
  if (document.querySelector('.cf-turnstile, iframe[src*="challenges.cloudflare.com"], #px-captcha, [id^="px-captcha"]')) return 'captcha';
  const p = location.pathname;
  if (/account-security|\/login\b|\/signup/i.test(p)) return 'login';
  if (/^\/nx\//.test(p) && window.__NUXT__) return 'in';
  return 'unknown';
}`

// AuthState reports whether the current (auth-gated) page shows a signed-in session.
func AuthState(page *rod.Page) Auth {
	obj, err := page.Eval(authJS)
	if err != nil {
		return AuthUnknown
	}
	switch Auth(obj.Value.Str()) {
	case AuthIn:
		return AuthIn
	case AuthLogin:
		return AuthLogin
	case AuthCaptcha:
		return AuthCaptcha
	default:
		return AuthUnknown
	}
}

// ProfileDir returns the profile directory this browser was launched with, for
// reporting to the user.
func (b *Browser) ProfileDir() string { return b.profileDir }
