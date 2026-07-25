# Scoring Guide — how to add and honestly rate a site

This guide exists so that two people rating the same site reach the **same
badge**. Ratings are only as trustworthy as the process behind them, so the
process is written down and enforced by tests.

## The golden rule: Unknown is honest

If you have **not personally verified** a signal, it is `unknown` — never a
guess and never an optimistic default. A site with three verified-clean signals
and four unknowns is `? Unclassified`, not `A`. Absence of evidence is never
evidence of privacy.

Concretely, the code guarantees this: an `unknown` signal can only ever *lower*
or fail to raise a grade, never raise it.

## The badge

Two parts, always rendered together. **Colour is never load-bearing** — use the
letter + name + icon so the badge is readable to everyone.

| Grade | Name | Icon | Meaning |
|-------|------|------|---------|
| A | Clean | `✓✓` | GPC honoured, no ad cookies, no third-party ads/trackers/scripts, nothing invasive — all verified |
| B | Considerate | `✓` | Verified clean on the disqualifiers, honours GPC, at most minor third-party content |
| C | Mixed | `~` | The honest middle: GPC not honoured, or some signals still unverified |
| D | Tracking | `!` | A confirmed disqualifier is present (ad cookies / fingerprinting / session replay / data selling) |
| F | Invasive | `✕` | A disqualifier **and** a governance failure (no GPC, or heavy trackers) |
| ? | Unclassified | `?` | Not enough verified signals to rate |

Plus a **trust marker** for provenance (separate axis):

| Marker | Trust | When to use |
|--------|-------|-------------|
| `★` | own | A first-party site under your own control. Set at runtime via `WithFirstParty()`, **not** in the dataset. |
| `✓` | audited | You personally verified the signals. |
| `~` | imported | Loaded from a build-time data snapshot. |
| (none) | unknown | Provenance not established. |

## The signals and how to observe each one

Open the site in a **fresh** browser profile (no extensions, cache bypassed) and
use DevTools. Record what you actually see, not what you expect.

| Field | How to verify | Value |
|-------|---------------|-------|
| `adTrackingCookies` | DevTools → Application → Cookies. Distinguish first-party functional cookies from ad-network ones (DoubleClick, `_ga`, `_fbp`, etc.). | `yes` if any ad/tracking cookie is set, else `no` |
| `honorsGPC` | Send `Sec-GPC: 1` / set `navigator.globalPrivacyControl`; check the privacy policy's GPC statement. | `yes` only if honouring is confirmed |
| `adsTrackers` | Network tab grouped by domain; count ad/tracker origins. | `none` / `some` (low) / `heavy` (high) |
| `thirdPartyScripts` | Network tab → JS from third-party origins. | `none` / `few` (low) / `many` (high) |
| `fingerprinting` | Look for canvas/WebGL/audio API probing (EFF *Cover Your Tracks* methodology). | `yes` if observed |
| `sessionReplay` | Network/scripts for Hotjar, FullStory, Clarity, mouseflow, etc. | `yes` if a recorder is present |
| `sellsSharesData` | Read the privacy policy + any "Do Not Sell/Share My Info" (CCPA) disclosure. | `yes` if they sell/share |
| `thirdPartyDomains` | Count of distinct third-party origins contacted. | integer (informational only) |

Aliases accepted in JSON: `some`/`few` → low, `heavy`/`many` → high, `true`/`false` → yes/no.

## How the grade is derived (worst-signal-dominates)

Evaluated in this order — the first matching rule wins:

1. **Own site** → `A`. (You control it.)
2. **Disqualifier confirmed** (`adTrackingCookies`, `fingerprinting`,
   `sessionReplay`, or `sellsSharesData` = `yes`) → capped at `D`.
3. **…and a governance failure** (`honorsGPC` = `no`, or `adsTrackers` = `high`)
   → escalates to `F`.
4. **Governance failure alone** (no disqualifier) → `C`.
5. **No confirmed-bad signals** → must be *earned*:
   - Requires `honorsGPC` = `yes` **and** `adTrackingCookies` = `no`. Otherwise
     → `Unclassified`.
   - Any remaining `unknown` among the other signals → `C`.
   - Fully verified clean, no ads/scripts → `A`; minor ads/scripts → `B`.

This is why you cannot game a good grade by leaving bad signals blank — blanks
can't lift you past `C`, and a confirmed disqualifier caps you no matter how
clean everything else is.

## Adding an entry

Add to your dataset JSON (the shipped seed is `data/domains.json`; deployers
usually load their own via `WithDataFile`). Key by registrable domain
(`example.com`) or an exact host (`sub.example.com`) when a subdomain differs.

```json
"trackingsite.example": {
  "trust": "audited",
  "verified": "2026-07-25",
  "note": "Short description shown with the badge.",
  "evidence": "What you observed: cookies X/Y, Hotjar present, no GPC.",
  "signals": {
    "adTrackingCookies": "yes",
    "honorsGPC": "no",
    "adsTrackers": "heavy",
    "thirdPartyScripts": "many",
    "fingerprinting": "unknown",
    "sessionReplay": "yes",
    "sellsSharesData": "yes",
    "thirdPartyDomains": 24
  }
}
```

Rules enforced by `New()` (and therefore by `go test`):

- Every `audited`/`imported` entry **must** have a valid `verified` date
  (`YYYY-MM-DD`), and it may not be in the future.
- Unknown JSON fields and invalid enum values are rejected — typos fail loudly.
- Entries older than the staleness window (default 1 year) are automatically
  demoted to `Unclassified` at classification time until re-verified. Sites
  change; a two-year-old rating is not trustworthy.

## Worked example — youtube.com → F

- `adTrackingCookies: yes` → disqualifier → cap at D.
- `honorsGPC: no` → governance failure → escalate to **F**.
- Reasons surfaced: *Sets ad/tracking cookies*, *Does not honour Global Privacy
  Control*, *Heavy third-party ads/trackers*, *Sells or shares personal data*.

Compare `youtube-nocookie.com`: cookies-on-playback are left `unknown` (not
asserted clean), but `honorsGPC: no` caps it at **C**. The real privacy win of
the `-nocookie` domain is *deferral*, which belongs to your page's click-to-load
facade, not to the destination's own grade — the rubric stays honest about that.
