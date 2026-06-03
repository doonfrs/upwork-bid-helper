# upwork-feed-fetcher

[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-%E2%98%95-orange.svg?style=flat-square)](https://buymeacoffee.com/doonfrs)

A command-line tool that exports your Upwork job feeds to **JSON** (or CSV/XML),
so a script or an AI model can read them, rank them, and decide what's worth
bidding on, without you (or an AI agent) clicking through a browser every time.

## Why

Browsing Upwork through an AI agent is slow and expensive. Every search, scroll,
and click burns screenshots and tokens. This tool flips it around:

1. Run it on a schedule (say, hourly). It opens Upwork in the background, grabs
   your feeds, and writes a clean JSON file.
2. Your script or AI reads that JSON, filters/ranks the jobs, and picks the good
   ones.
3. You only open the browser when there's actually a job worth bidding on.

Fast, cheap, and easy to automate.

## What it exports

Every job comes out as structured data. The exact schema (JSON field names) is below.

**Job**

| field | type | notes |
|-------|------|-------|
| `id` | string | ciphertext id, e.g. `~021234…` |
| `uid` | string | numeric id |
| `recno` | string | Upwork record number |
| `url` | string | derived from `id` |
| `title` | string | |
| `description` | string | |
| `type` | string | `hourly` or `fixed` |
| `hourlyMin` / `hourlyMax` | number | hourly range (0 if fixed) |
| `fixedBudget` | number | fixed price (0 if hourly) |
| `weeklyBudget` | number | rarely set |
| `engagement` | string | e.g. `Less than 30 hrs/week` |
| `duration` | string | e.g. `More than 6 months` |
| `experienceLevel` | string | `Entry Level` / `Intermediate` / `Expert` |
| `freelancersToHire` | int | |
| `proposalsTier` | string | bucket, e.g. `5 to 10` |
| `totalApplicants` | int | exact count — **my-feed only** |
| `premium` / `applied` / `enterprise` | bool | |
| `jobStatus` | string | e.g. `Open` — **my-feed only** |
| `isLocal` | bool | **my-feed only** |
| `prefFreelancerLocation` | string[] | preferred countries — **my-feed only** |
| `prefFreelancerLocationMandatory` | bool | |
| `createdOn` / `publishedOn` | string | ISO 8601 |
| `renewedOn` | string | ISO 8601 — **my-feed only** |
| `connectPrice` | int | connects to apply |
| `position` | int | rank within the feed |
| `skills` | string[] | |
| `tags` | string[] | e.g. `firstJobPost`, `contractToHireSet` |
| `client` | object | see below |

**Client**

| field | type | notes |
|-------|------|-------|
| `paymentVerified` | bool | |
| `totalSpent` | number | USD across all contracts |
| `totalReviews` | int | |
| `rating` | number | 0–5 |
| `totalHires` | int | |
| `totalPostedJobs` | int | **my-feed only** |
| `country` | string | name or ISO-3 code (Upwork is inconsistent) |
| `city` | string | usually empty in feeds |
| `topClient` | bool | |
| `financialPrivacy` | bool | client hides spend/reviews |
| `lastRecruitingActivity` | string | ISO 8601; empty if none |
| `companyOrgUid` | string | stable client-org id — **my-feed only** |

> **Feeds use two shapes.** `most-recent` and `best` send a *lean* payload;
> `my-feed` sends a *richer* one. Fields marked **my-feed only** above are absent
> from `recent`/`best` exports and come out as `0` / `""` / `[]` there. In `all`
> mode, jobs are deduplicated keeping the `myfeed` copy first, so shared jobs keep
> the richer fields.

> **What's *not* available:** the client's **average hourly rate** and **total
> hours billed** are not in any feed payload — Upwork only shows them on the
> single job page, which sits behind a Cloudflare bot wall the automated browser
> can't pass. So those two fields can't be exported.

It can export any of your Find Work feeds:

| command  | feed                                              |
|----------|---------------------------------------------------|
| `myfeed` | your personalized feed (from your saved searches) |
| `best`   | Best Matches                                      |
| `recent` | Most Recent                                       |
| `saved`  | Saved (bookmarked) jobs                           |

…or a single job from its URL.

> **No keyword search.** Upwork puts the search page behind a Cloudflare bot
> challenge that blocks automated browsers, so search isn't supported. The feeds
> above are *not* blocked. Set up Saved Searches on Upwork (they power `myfeed`)
> to get keyword-targeted results.

## Requirements

- Go 1.23+
- Google Chrome

## Build

```sh
go build -o upwork-feed-fetcher ./cmd/upwork-feed-fetcher
```

On Windows:

```powershell
go build -o upwork-feed-fetcher.exe ./cmd/upwork-feed-fetcher
```

## Usage

### 1. Sign in once

```sh
upwork-feed-fetcher login
```

Opens a visible Chrome. Sign in to Upwork (and solve any CAPTCHA). Your session
is saved and reused on every later run; you won't sign in again unless it expires.

### 2. Export a feed

```sh
upwork-feed-fetcher myfeed     # your personalized feed
upwork-feed-fetcher best       # best matches
upwork-feed-fetcher recent     # most recent
upwork-feed-fetcher saved      # saved jobs
```

These run in the background (no window) and write a file like
`upwork-feed-20260603-2130.json`.

### 3. Export everything at once

```sh
upwork-feed-fetcher all
```

Sweeps **myfeed + best + recent** in one run, merges them, removes duplicates, and
writes a single `upwork-all-….json`. This is the one to schedule.

### 4. Load more jobs per feed

You get the first page by default. To pull more, click "Load More" automatically:

```sh
upwork-feed-fetcher all --pages 2      # ~2 pages per feed
upwork-feed-fetcher recent --pages 3   # 3 pages
```

(Best Matches is a fixed list with no "Load More", so `--pages` only adds jobs to
`myfeed` and `recent`.)

### 5. A single job

```sh
upwork-feed-fetcher "https://www.upwork.com/jobs/~021234567890abcdef"
```

### Choosing the output

```sh
upwork-feed-fetcher all --output jobs.json      # name the file
upwork-feed-fetcher all --format csv            # or csv / xml
upwork-feed-fetcher all --format json,csv,xml   # several at once
```

With no `--output`, it writes `upwork-<type>-<timestamp>.json` in the current
folder.

## Options

| flag | default | what it does |
|------|---------|--------------|
| `--output` / `--out` | auto | output file (or name prefix when multiple formats) |
| `--format` | `json` | `json`, `csv`, `xml`, `all`, or a comma list like `json,csv` |
| `--pages` | `1` | pages to load per feed (clicks "Load More" `pages−1` times) |
| `--gui` | off | show the browser window (handy to watch a run) |
| `--hold` | off | do the action, then keep the window open until Ctrl+C (manual poking) |
| `--timeout` | `90s` | how long to wait for a page to load |
| `--dry-run` | off | print the URL(s) it would visit and exit (doesn't open the browser) |
| `--raw` | off | dump the untouched client/job payload from the page as JSON and exit (diagnostic; needs a single feed/job, no export) |
| `--profile` | app data dir | where the saved login/profile is kept |
| `--chrome` | system Chrome | path to a specific Chrome binary |

## Example: hourly triage

```sh
upwork-feed-fetcher all --pages 2 --output jobs.json
# → hand jobs.json to Claude / Codex / your script to rank and decide,
#   then open Upwork only for the jobs worth bidding on.
```

## Tests

```sh
go test ./...
```

## Support

If this tool saves you time, here's how to help keep it going:

- ⭐ **Star the repo** on GitHub
- ☕ **Buy me a coffee**: [buymeacoffee.com/doonfrs](https://buymeacoffee.com/doonfrs)

Every star and coffee means a lot and helps maintain the project! 🚀

## Author

**Feras Abdalrahman**

- GitHub: [@doonfrs](https://github.com/doonfrs)
- LinkedIn: [in/doonfrs](https://www.linkedin.com/in/doonfrs/)

💼 **Available for freelance work.** Need a developer for your project? Reach out
at [doonfrs@gmail.com](mailto:doonfrs@gmail.com).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file
for details.
