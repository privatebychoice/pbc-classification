package classify

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// defaultData is the curated seed dataset shipped with the module. It is loaded
// unless WithoutDefaultData() is passed to New.
//
//go:embed data/domains.json
var defaultData []byte

// Entry is a single domain's record in the dataset.
type Entry struct {
	// Trust is the provenance of this record.
	Trust Trust `json:"trust,omitempty"`
	// Signals is what has been verified about the domain's behaviour.
	Signals Signals `json:"signals,omitempty"`
	// Verified is the ISO date (YYYY-MM-DD) the record was last checked.
	// Required for audited/imported entries so stale ratings can be detected.
	Verified string `json:"verified,omitempty"`
	// Evidence is a short human note describing what was observed.
	Evidence string `json:"evidence,omitempty"`
	// Note is a free-form comment shown alongside the badge.
	Note string `json:"note,omitempty"`
}

// parseDataset decodes a JSON object of {domain: Entry}. Unknown fields are
// rejected so typos in hand-authored data fail loudly rather than silently
// producing an over-optimistic rating.
func parseDataset(b []byte) (map[string]Entry, error) {
	m := map[string]Entry{}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("classify: parsing dataset: %w", err)
	}
	return m, nil
}

// mergeEntries overlays src onto dst, lower-casing keys. Later sources override
// earlier ones by domain, so runtime overrides beat the embedded defaults.
func mergeEntries(dst, src map[string]Entry) {
	for k, v := range src {
		dst[strings.ToLower(strings.TrimSpace(k))] = v
	}
}

// validateEntries enforces the dataset invariants that keep ratings honest:
// audited/imported entries must carry a valid, non-future verification date.
// (Enum validity is already enforced during JSON decoding.) Own entries are
// operator-asserted and need no date.
func validateEntries(m map[string]Entry, now func() time.Time) error {
	for host, e := range m {
		if e.Trust == TrustOwn {
			continue
		}
		if e.Verified == "" {
			return fmt.Errorf("classify: entry %q is %s but has no 'verified' date", host, e.Trust)
		}
		t, err := time.Parse("2006-01-02", e.Verified)
		if err != nil {
			return fmt.Errorf("classify: entry %q has invalid 'verified' date %q (want YYYY-MM-DD)", host, e.Verified)
		}
		if t.After(now()) {
			return fmt.Errorf("classify: entry %q has a future 'verified' date %q", host, e.Verified)
		}
	}
	return nil
}
