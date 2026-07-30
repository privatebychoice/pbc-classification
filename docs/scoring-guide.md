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
| A | Clean | `✓✓` | Fully verified clean (no third-party ad cookies, ads/trackers, scripts, or anything invasive) **and** honours GPC |
| B | Considerate | `✓` | Confirmed no third-party ad cookies and at most minor third-party content — reached even when GPC is unverified |
| C | Mixed | `~` | The honest middle: some signals verified but not enough to confirm clean, or an in-scope site that doesn't honour GPC |
| D | Tracking | `!` | A confirmed disqualifier is present (third-party ad cookies / fingerprinting / session replay / data selling) |
| F | Invasive | `✕` | A disqualifier **and** a governance failure (heavy trackers, or ad-tracking/selling without honouring GPC) |
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
| `adTrackingCookies` | DevTools → Application → Cookies. Only **third-party / cross-site advertising or tracking** cookies count (DoubleClick, `_fbp`, ad-network `_ga` linkage, etc.). Benign first-party functional or privacy-respecting first-party analytics cookies do **not** count. | `yes` only for third-party ad/tracking cookies; a site with only first-party functional cookies is `no` |
| `honorsGPC` | Send `Sec-GPC: 1` / set `navigator.globalPrivacyControl`; check the privacy policy's GPC statement. Only meaningful for sites that sell/share or ad-track. | `yes` only if honouring is confirmed. Record `no` **only** for a site that sells/shares or ad-tracks; otherwise leave it Unknown (there is nothing to honour). |
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
3. **…and a governance failure** → escalates to `F`. A governance failure is
   `adsTrackers` = `high`, **or** `honorsGPC` = `no` *when GPC is applicable* —
   i.e. the site sells/shares or ad-tracks (`adTrackingCookies` = `yes` or
   `adsTrackers` = `low`/`high`).
4. **Governance failure alone** (no disqualifier) → `C`.
5. **No confirmed-bad signals** → earned from positive evidence:
   - `adTrackingCookies` = `no` **and** at most minor third-party content
     (`adsTrackers` `none`/`low` and `thirdPartyScripts` not `high`) → `B`
     **even when `honorsGPC` is Unknown**.
   - The same site that *also* honours GPC and is fully verified clean
     (`adsTrackers` `none`, `thirdPartyScripts` `none`, and `fingerprinting`,
     `sessionReplay`, `sellsSharesData` all `no`) → `A`.
   - Otherwise, if any behavioural signal is verified but the site isn't
     confirmed clean → `C`; if essentially nothing is verified → `Unclassified`.

**On `honorsGPC`.** It is a *booster*, not a gate. Honouring GPC lifts a
confirmed-clean site to `A`; leaving it Unknown does **not** strand a clean site
at `?`. And GPC honouring only matters for a site that has something to opt out
of — for a site that neither sells/shares nor ad-tracks, `honorsGPC` = `no` is
inert (it never caps the grade, and never scores better than Unknown).

Two invariants the code guarantees: an `unknown` signal never *raises* a grade,
and recording `honorsGPC` = `no` never yields a *better* grade than leaving it
Unknown. You cannot game a good grade by leaving bad signals blank — a confirmed
disqualifier caps you no matter how clean everything else looks, and `A` requires
full positive confirmation **plus** GPC.

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

- `adTrackingCookies: yes` (third-party) → disqualifier → cap at D.
- The site ad-tracks and sells/shares, so GPC is applicable; `honorsGPC: no`
  (plus `adsTrackers: high`) is a governance failure → escalate to **F**.
- Reasons surfaced: *Sets third-party ad/tracking cookies*, *Sells or shares
  personal data*, *Heavy third-party ads/trackers*, *Sells/shares or ad-tracks
  without honouring Global Privacy Control*.

Compare `youtube-nocookie.com`: cookies-on-playback are left `unknown` (not
asserted clean), so there is no disqualifier — but it still ad-tracks
(`adsTrackers: low`), which makes GPC applicable, and `honorsGPC: no` is then a
governance failure that caps it at **C**. The real privacy win of the `-nocookie`
domain is *deferral*, which belongs to your page's click-to-load facade, not to
the destination's own grade — the rubric stays honest about that.

Contrast a clean reference site (e.g. `signal.org`): no third-party ad cookies,
no ads/trackers, doesn't sell/share, `honorsGPC` unverified. It now grades **B**
"Considerate" rather than `?` — a confirmed-clean site is no longer penalised for
unverified GPC. Confirm GPC honouring to lift it to **A**.
