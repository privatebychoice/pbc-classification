package classify

import (
	"fmt"
	"strings"
)

// Grade is the end-user-facing privacy badge. It is deliberately ordinal so a
// higher constant is a better grade, but never compare grades by magnitude in
// UI — always render Letter + Name + Icon so nothing depends on colour.
type Grade uint8

const (
	// GradeUnclassified: not enough verified signals to rate honestly.
	GradeUnclassified Grade = iota
	GradeF
	GradeD
	GradeC
	GradeB
	GradeA
)

// Letter returns the single-character grade (A–F, or "?" when unclassified).
func (g Grade) Letter() string {
	switch g {
	case GradeA:
		return "A"
	case GradeB:
		return "B"
	case GradeC:
		return "C"
	case GradeD:
		return "D"
	case GradeF:
		return "F"
	default:
		return "?"
	}
}

// Name returns the human-readable label for the grade.
func (g Grade) Name() string {
	switch g {
	case GradeA:
		return "Clean"
	case GradeB:
		return "Considerate"
	case GradeC:
		return "Mixed"
	case GradeD:
		return "Tracking"
	case GradeF:
		return "Invasive"
	default:
		return "Unclassified"
	}
}

// Icon returns a short, shape-distinct token for the grade. The glyph alone
// distinguishes each grade so colour is never load-bearing (accessibility).
// A consuming UI is expected to map these to its own self-hosted SVG icons.
func (g Grade) Icon() string {
	switch g {
	case GradeA:
		return "✓✓"
	case GradeB:
		return "✓"
	case GradeC:
		return "~"
	case GradeD:
		return "!"
	case GradeF:
		return "✕"
	default:
		return "?"
	}
}

func (g Grade) String() string { return g.Letter() + " " + g.Name() }

// Trust is the provenance axis: how much the rating's *source* can be trusted.
// It is separate from the behaviour Grade and is rendered as a small marker
// alongside it.
type Trust uint8

const (
	TrustUnknown Trust = iota
	// TrustImported: from a data snapshot loaded at build time.
	TrustImported
	// TrustAudited: the operator personally verified the signals.
	TrustAudited
	// TrustOwn: a first-party site under the operator's own control.
	TrustOwn
)

func (t Trust) String() string {
	switch t {
	case TrustImported:
		return "imported"
	case TrustAudited:
		return "audited"
	case TrustOwn:
		return "own"
	default:
		return "unknown"
	}
}

// Marker returns a short provenance glyph (or "" for unknown provenance).
func (t Trust) Marker() string {
	switch t {
	case TrustOwn:
		return "★"
	case TrustAudited:
		return "✓"
	case TrustImported:
		return "~"
	default:
		return ""
	}
}

func (t *Trust) UnmarshalText(b []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(b))) {
	case "", "unknown":
		*t = TrustUnknown
	case "imported":
		*t = TrustImported
	case "audited":
		*t = TrustAudited
	case "own":
		*t = TrustOwn
	default:
		return fmt.Errorf("classify: invalid trust %q (want own|audited|imported|unknown)", string(b))
	}
	return nil
}

func (t Trust) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

// grade applies the "worst-signal-dominates" rubric and returns the resulting
// grade plus the human-readable reasons that justify it. The rules, in order:
//
//   - A first-party (Own) site is always A — the operator controls it.
//   - Disqualifiers (confirmed ad cookies, fingerprinting, session replay, or
//     data selling/sharing) cap the grade at D "Tracking".
//   - Governance failures (does not honour GPC, or heavy ads/trackers) escalate
//     a disqualified site to F "Invasive", or on their own cap a site at C.
//   - With no confirmed-bad signals, a top grade must be *earned* by positive
//     confirmation: GPC honoured and no ad cookies. Any remaining Unknown holds
//     the site at C; a fully-verified-clean profile is A (or B if it loads
//     minor third-party content). Absence of evidence is never a pass.
func grade(s Signals, trust Trust) (Grade, []string) {
	if trust == TrustOwn {
		return GradeA, []string{"First-party site under the operator's own control"}
	}

	var reasons []string

	// Disqualifiers — any one confirmed caps the grade at D.
	disq := false
	if s.AdTrackingCookies == Yes {
		disq = true
		reasons = append(reasons, "Sets ad/tracking cookies")
	}
	if s.Fingerprinting == Yes {
		disq = true
		reasons = append(reasons, "Uses browser fingerprinting")
	}
	if s.SessionReplay == Yes {
		disq = true
		reasons = append(reasons, "Records sessions (mouse/keystroke replay)")
	}
	if s.SellsSharesData == Yes {
		disq = true
		reasons = append(reasons, "Sells or shares personal data")
	}

	// Governance failures — escalate a disqualified site to F, or on their own
	// hold an otherwise-unknown site at C.
	gov := false
	if s.HonorsGPC == No {
		gov = true
		reasons = append(reasons, "Does not honour Global Privacy Control")
	}
	if s.AdsTrackers == LevelHigh {
		gov = true
		reasons = append(reasons, "Heavy third-party ads/trackers")
	}

	switch {
	case disq && gov:
		return GradeF, reasons
	case disq:
		return GradeD, reasons
	case gov:
		return GradeC, reasons
	}

	// No confirmed-bad signals. Require positive confirmation to pass — absence
	// of evidence is not a pass.
	if s.HonorsGPC != Yes || s.AdTrackingCookies != No {
		reasons = append(reasons, "Not enough verified signals to rate")
		return GradeUnclassified, reasons
	}
	reasons = append(reasons, "Honours GPC", "No ad/tracking cookies observed")

	// GPC confirmed and cookies confirmed absent. Any remaining Unknown keeps us
	// at the honest middle rather than awarding a top grade.
	if s.AdsTrackers == LevelUnknown || s.ThirdPartyScripts == LevelUnknown ||
		s.Fingerprinting == Unknown || s.SessionReplay == Unknown || s.SellsSharesData == Unknown {
		reasons = append(reasons, "Some signals not yet verified")
		return GradeC, reasons
	}

	if s.AdsTrackers == LevelNone && s.ThirdPartyScripts == LevelNone {
		return GradeA, reasons
	}
	reasons = append(reasons, "Loads some third-party content")
	return GradeB, reasons
}
