// Package classify is a lightweight, offline privacy-badge classifier for the
// external links used on a website. Given a URL it resolves the registrable
// domain, looks the domain up in a locally-curated dataset (no third-party
// calls at runtime), and returns an easy-to-understand badge — a letter Grade
// with a Name and Icon — plus the machine-readable Signals and the
// human-readable Reasons behind it.
//
// The grading rubric is documented in docs/scoring-guide.md and implemented in
// grade.go. Its guiding principle is honesty: an unverified signal is Unknown
// and can never raise a grade, and any single confirmed-bad signal caps the
// result (worst-signal-dominates).
package classify

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// defaultStaleAfter is how long an audited/imported rating is trusted before it
// is treated as stale and demoted to Unclassified until re-verified.
const defaultStaleAfter = 365 * 24 * time.Hour

// Classifier holds a compiled dataset and classifies URLs against it. It is safe
// for concurrent use after construction.
type Classifier struct {
	entries    map[string]Entry
	staleAfter time.Duration
	now        func() time.Time
}

// Classification is the result of classifying a single URL.
type Classification struct {
	Input    string   // the URL that was classified
	Domain   string   // the registrable domain (or host) it resolved to
	Matched  bool     // whether a dataset entry was found
	Grade    Grade    // the behaviour badge
	Trust    Trust    // provenance of the rating
	Signals  Signals  // the signals the grade was derived from
	Reasons  []string // human-readable justification for the grade
	Verified string   // ISO date the record was last verified
	Stale    bool     // true if the record is past the staleness threshold
	Note     string   // free-form note from the dataset entry
}

func (c Classification) String() string {
	m := c.Grade.Icon()
	if mk := c.Trust.Marker(); mk != "" {
		m = mk + " " + m
	}
	return fmt.Sprintf("%s [%s]", c.Grade, strings.TrimSpace(m))
}

type config struct {
	staleAfter   time.Duration
	now          func() time.Time
	omitDefaults bool
	sources      []func() (map[string]Entry, error)
}

// Option configures a Classifier.
type Option func(*config)

// WithoutDefaultData skips the embedded seed dataset, starting from an empty set.
func WithoutDefaultData() Option { return func(c *config) { c.omitDefaults = true } }

// WithStaleAfter sets how long an audited/imported rating is trusted before it
// is demoted to Unclassified. A non-positive duration disables staleness.
func WithStaleAfter(d time.Duration) Option { return func(c *config) { c.staleAfter = d } }

// WithClock overrides the time source (used in tests and for deterministic
// staleness). A nil argument is ignored.
func WithClock(now func() time.Time) Option {
	return func(c *config) {
		if now != nil {
			c.now = now
		}
	}
}

// WithDataBytes merges an additional JSON dataset (overriding earlier entries).
func WithDataBytes(b []byte) Option {
	return func(c *config) {
		c.sources = append(c.sources, func() (map[string]Entry, error) { return parseDataset(b) })
	}
}

// WithDataFile merges an additional JSON dataset read from path at construction.
func WithDataFile(path string) Option {
	return func(c *config) {
		c.sources = append(c.sources, func() (map[string]Entry, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("classify: reading %q: %w", path, err)
			}
			return parseDataset(b)
		})
	}
}

// WithFirstParty registers one or more domains as first-party (Trust Own, always
// grade A). This is how a deployer marks their *own* sites — first-party trust is
// per-deployment and is never shipped in the dataset, so nobody inherits trust
// in another operator's sites.
func WithFirstParty(domains ...string) Option {
	return func(c *config) {
		c.sources = append(c.sources, func() (map[string]Entry, error) {
			m := make(map[string]Entry, len(domains))
			for _, d := range domains {
				key, err := registrableDomain(d)
				if err != nil {
					return nil, fmt.Errorf("classify: first-party %q: %w", d, err)
				}
				m[key] = Entry{Trust: TrustOwn, Note: "Configured first-party site"}
			}
			return m, nil
		})
	}
}

// New builds a Classifier. By default it loads the embedded seed dataset, then
// applies each data source in order (later sources override earlier ones), then
// validates the merged dataset.
func New(opts ...Option) (*Classifier, error) {
	cfg := &config{staleAfter: defaultStaleAfter, now: time.Now}
	for _, o := range opts {
		o(cfg)
	}

	entries := map[string]Entry{}
	if !cfg.omitDefaults {
		def, err := parseDataset(defaultData)
		if err != nil {
			return nil, fmt.Errorf("classify: loading default dataset: %w", err)
		}
		mergeEntries(entries, def)
	}
	for _, src := range cfg.sources {
		m, err := src()
		if err != nil {
			return nil, err
		}
		mergeEntries(entries, m)
	}
	if err := validateEntries(entries, cfg.now); err != nil {
		return nil, err
	}

	return &Classifier{entries: entries, staleAfter: cfg.staleAfter, now: cfg.now}, nil
}

// Classify resolves rawURL to a domain and returns its privacy classification.
// An unparseable URL or an unknown domain yields GradeUnclassified rather than
// an error — the honest answer is "we don't know", not a failure.
func (c *Classifier) Classify(rawURL string) Classification {
	res := Classification{Input: rawURL}

	host := hostOf(rawURL)
	if host == "" {
		res.Grade = GradeUnclassified
		res.Reasons = []string{"Could not parse a hostname from the URL"}
		return res
	}

	entry, key, ok := c.lookup(host)
	res.Domain = key
	if !ok {
		res.Grade = GradeUnclassified
		res.Reasons = []string{"No classification on record for this domain"}
		return res
	}

	res.Matched = true
	res.Trust = entry.Trust
	res.Signals = entry.Signals
	res.Verified = entry.Verified
	res.Note = entry.Note

	// A stale audited/imported rating is demoted to Unclassified until refreshed;
	// first-party (Own) sites are operator-controlled and never go stale.
	if entry.Trust != TrustOwn && c.isStale(entry) {
		res.Stale = true
		res.Grade = GradeUnclassified
		res.Reasons = []string{staleReason(entry.Verified)}
		return res
	}

	res.Grade, res.Reasons = grade(entry.Signals, entry.Trust)
	return res
}

func staleReason(verified string) string {
	if verified == "" {
		return "Rating has no verification date; re-verify before trusting"
	}
	return fmt.Sprintf("Rating is stale (last verified %s); re-verify before trusting", verified)
}

// lookup finds the entry for a host: an exact host match wins (so a specific
// subdomain can be rated on its own), otherwise the registrable domain is used.
func (c *Classifier) lookup(host string) (Entry, string, bool) {
	if e, ok := c.entries[host]; ok {
		return e, host, true
	}
	reg, err := registrableDomain(host)
	if err != nil {
		return Entry{}, host, false
	}
	if e, ok := c.entries[reg]; ok {
		return e, reg, true
	}
	return Entry{}, reg, false
}

func (c *Classifier) isStale(e Entry) bool {
	if c.staleAfter <= 0 {
		return false
	}
	if e.Verified == "" {
		return true
	}
	t, err := time.Parse("2006-01-02", e.Verified)
	if err != nil {
		return true
	}
	return c.now().Sub(t) > c.staleAfter
}

// hostOf extracts a lower-cased hostname from a URL that may lack a scheme.
func hostOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "//" + raw // let url.Parse treat a bare host as authority
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
}

// registrableDomain returns the eTLD+1 (public-suffix-aware) for a host, so
// multi-part TLDs like "co.uk" are handled correctly.
func registrableDomain(host string) (string, error) {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return "", fmt.Errorf("empty host")
	}
	// Accept a full URL or bare host.
	if strings.Contains(host, "/") || strings.Contains(host, ":") {
		host = hostOf(host)
		if host == "" {
			return "", fmt.Errorf("no host")
		}
	}
	d, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return "", err
	}
	return d, nil
}
