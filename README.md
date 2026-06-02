# upwork-bid-helper

Drives a real Chrome (via [go-rod](https://github.com/go-rod/rod)) to export Upwork
**feed**, **search**, or **single-job** pages to **JSON / CSV / XML**. It reads the
data the page already loaded (`window.__NUXT__` / the Nuxt3 `__NUXT_DATA__` payload),
so there is no fragile HTML scraping.

> Status: **Step 1 — CLI vertical slice.** GUI mode (double-click), hidden/off-screen
> mode, network interception, and rich single-job client data are upcoming steps.

## Build

```sh
go build -o upwork-bid-helper ./cmd/upwork-bid-helper
```

Requires Go 1.23+ and Google Chrome installed.

## Usage

```sh
# A full Upwork URL (feed, search, or a job page):
upwork-bid-helper "https://www.upwork.com/nx/find-work/most-recent"
upwork-bid-helper "https://www.upwork.com/jobs/~02xxxxxxxxxxxxxxxxx"

# Or build a search from key=value args (bare words become the query):
upwork-bid-helper q="react native" payment_verified=1
```

The browser opens **visibly**. On first run, log in (and solve any CAPTCHA) in that
window — the session is saved to a persistent profile and reused on later runs. If a
login/challenge appears mid-run the tool waits for you to finish, then continues.

### Flags

| flag | default | meaning |
|------|---------|---------|
| `--format` | `all` | `json` \| `csv` \| `xml` \| `all` |
| `--out` | auto | output file, or filename prefix when `--format=all` |
| `--chrome` | system Chrome | path to a Chrome binary |
| `--profile` | app config dir | persistent profile directory |
| `--timeout` | `3m` | max wait for the page (includes manual login) |
| `--headless` | off | **local `file://` exports/testing only** — never use against live Upwork (instantly bot-flagged) |

`file://` paths are accepted as targets so you can export from a saved page offline.

## Test

```sh
go test ./...
```

The extractor tests load the saved samples in `temp/` (a local, gitignored scratch
dir) in headless Chrome and assert the feed/search/single-job extraction.

## Layout

- `cmd/upwork-bid-helper` — CLI entrypoint
- `internal/browser` — launch, persistent profile, challenge detection, teardown
- `internal/extract` — page-type detection + `window.__NUXT__`/devalue extractor (`extract.js`)
- `internal/model` — normalized `Job` / `Client` / `Result`
- `internal/export` — JSON / CSV (formula-injection guarded) / XML (escaped)
- `internal/search` — `key=value` → search URL builder
