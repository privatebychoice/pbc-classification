# pbc-classification

**Privacy Badge Classification** — a lightweight, **offline** privacy-badge
classifier for the external links on a website. Give it a URL; it returns an
easy-to-understand badge (a letter grade with a name and icon), plus the
machine-readable signals and human-readable reasons behind it. No third-party
calls at runtime; all data is local and configurable.

> `pbc` here = **Privacy Badge Classification** (this library). It is a distinct
> project and is **not** the Privacy By Choice brand.

## Install

```bash
go get go.privatebychoice.com/pbc-classification
```

## Use

```go
import classify "go.privatebychoice.com/pbc-classification"

c, err := classify.New(
    // Your own sites — first-party trust is per-deployment, never shipped.
    classify.WithFirstParty("privatebychoice.com", "theuntrackedlife.com"),
    // Optional: merge your own curated dataset (overrides the seed).
    classify.WithDataFile("privacy-data.json"),
)
if err != nil {
    log.Fatal(err)
}

r := c.Classify("https://www.youtube.com/watch?v=abc")
fmt.Println(r.Grade.Letter(), r.Grade.Name(), r.Grade.Icon()) // F Invasive ✕
fmt.Println(r.Reasons) // why it got that grade
```

## The rating system

| Grade | Name | Meaning |
|-------|------|---------|
| A | Clean | Verified clean; honours GPC; no ad cookies, ads, trackers, or third-party scripts |
| B | Considerate | Verified clean on the disqualifiers; honours GPC; minor third-party content |
| C | Mixed | GPC not honoured, or some signals unverified |
| D | Tracking | A confirmed disqualifier (ad cookies / fingerprinting / session replay / data selling) |
| F | Invasive | A disqualifier plus a governance failure (no GPC, or heavy trackers) |
| ? | Unclassified | Not enough verified signals to rate honestly |

Plus a provenance marker: `★` own · `✓` audited · `~` imported.

**Signals:** ad/tracking cookies · honours GPC · ads & trackers · third-party
scripts · fingerprinting · session replay · sells/shares data · third-party
domain count.

The grade is derived **worst-signal-dominates**: an unverified signal is
`unknown` and can never raise a grade, and any single confirmed-bad signal caps
the result. See [`docs/scoring-guide.md`](docs/scoring-guide.md) for how to add
and honestly rate a site.

## Design principles

- **Offline & configurable** — no runtime third-party calls; ship your own data.
- **Honest by construction** — `unknown` never passes as clean; stale ratings
  (default > 1 year) auto-demote to Unclassified until re-verified.
- **Accessible** — every badge is letter + name + icon; colour is never
  load-bearing.
- **Reusable** — first-party trust is configured per deployment, so nobody
  inherits trust in another operator's sites.

## CLI

A small reference command lives in [`cmd/classify`](cmd/classify) — handy for
spot-checking domains and as a known-good example of using the library.

```bash
go run ./cmd/classify "https://www.youtube.com/watch?v=abc" https://example.com
go run ./cmd/classify -json https://youtube.com                 # machine-readable
go run ./cmd/classify -first-party privatebychoice.com https://privatebychoice.com
cat urls.txt | go run ./cmd/classify                            # batch from stdin
```

The `-json` output is a preview of what an SSG per-page privacy manifest entry
could look like.

## Development

Common tasks live in the `Makefile` (requires Go 1.26+):

```bash
make check   # fmt-check + vet + test + govulncheck — run before pushing
make help    # list every target (race, cover, tidy, ...)
```

`make check` is the local quality gate. `govulncheck` runs via `go run` and is
never added to `go.mod`. GitHub Actions CI is planned separately.

## Testing

```bash
go test ./...
```

The tests include a guardrail that fails the build on malformed dataset entries
(missing/invalid/future verification dates, unknown fields, invalid enums).
