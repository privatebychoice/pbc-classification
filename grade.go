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
//   - Disqualifiers (confirmed third-party ad cookies, fingerprinting, session
//     replay, or data selling/sharing) cap the grade at D "Tracking".
//   - Governance failures escalate a disqualified site to F "Invasive", or on
//     their own cap a site at C. Heavy ads/trackers are always a governance
//     failure; not honouring GPC is one only when GPC is *applicable* — i.e. the
//     site actually sells/shares or runs ad-tracking. A site with nothing to opt
//     out of has nothing to honour.
//   - With no confirmed-bad signals, the grade is earned from positive evidence:
//     confirmed no third-party ad cookies with at most minor third-party content
//     is B "Considerate" even when GPC is Unknown, and honouring GPC on a
//     fully-verified-clean site lifts it to A "Clean". honorsGPC is a booster,
//     never a gate.
//
// Two invariants hold throughout: an Unknown signal never *raises* a grade, and
// recording honorsGPC=No never yields a better grade than leaving it Unknown.
func grade(s Signals, trust Trust) (Grade, []string) {
	if trust == TrustOwn {
		return GradeA, []string{"First-party site under the operator's own control"}
	}

	var reasons []string

	// Disqualifiers — any one confirmed caps the grade at D. adTrackingCookies
	// means third-party / cross-site advertising or tracking cookies, not benign
	// first-party functional cookies (see docs/scoring-guide.md).
	disq := false
	if s.AdTrackingCookies == Yes {
		disq = true
		reasons = append(reasons, "Sets third-party ad/tracking cookies")
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

	// GPC only means something for a site that sells/shares or runs ad-tracking —
	// a site with nothing to opt out of has nothing to honour. So honorsGPC=No is
	// a governance failure only when GPC is applicable; otherwise it is inert, and
	// so never scores worse — or better — than leaving honorsGPC Unknown.
	adTracking := s.AdTrackingCookies == Yes ||
		s.AdsTrackers == LevelLow || s.AdsTrackers == LevelHigh
	gpcApplicable := s.SellsSharesData == Yes || adTracking

	// Governance failures — escalate a disqualified site to F, or on their own
	// cap a site at C.
	gov := false
	if s.AdsTrackers == LevelHigh {
		gov = true
		reasons = append(reasons, "Heavy third-party ads/trackers")
	}
	if s.HonorsGPC == No && gpcApplicable {
		gov = true
		reasons = append(reasons, "Sells/shares or ad-tracks without honouring Global Privacy Control")
	}

	switch {
	case disq && gov:
		return GradeF, reasons
	case disq:
		return GradeD, reasons
	case gov:
		return GradeC, reasons
	}

	// No confirmed-bad signals. Earn the grade from positive evidence; Unknown
	// signals hold a site below the fully-confirmed tiers but never sink a
	// confirmed-clean site to the bottom.
	noAdCookies := s.AdTrackingCookies == No
	fullyClean := noAdCookies &&
		s.SellsSharesData == No &&
		s.Fingerprinting == No &&
		s.SessionReplay == No &&
		s.AdsTrackers == LevelNone &&
		s.ThirdPartyScripts == LevelNone

	switch {
	case s.HonorsGPC == Yes && fullyClean:
		reasons = append(reasons,
			"No third-party ad cookies, no third-party ads/trackers, and honours GPC")
		return GradeA, reasons
	case noAdCookies && s.AdsTrackers != LevelUnknown && s.ThirdPartyScripts != LevelHigh:
		// Confirmed no third-party ad cookies and no worse-than-minor third-party
		// content: B even when GPC is Unknown (a confirmed-clean site is not
		// penalised for unverified GPC; only honouring GPC reaches A).
		if s.HonorsGPC == Yes {
			reasons = append(reasons,
				"No third-party ad cookies, only minor third-party content, and honours GPC")
		} else {
			reasons = append(reasons,
				"No third-party ad cookies and only minor third-party content")
		}
		return GradeB, reasons
	}

	// Some behavioural signals verified, but not enough to confirm a clean site.
	if hasCleanlinessEvidence(s) {
		reasons = append(reasons, "Some signals verified, but not enough to confirm a clean site")
		return GradeC, reasons
	}

	reasons = append(reasons, "Not enough verified signals to rate")
	return GradeUnclassified, reasons
}

// hasCleanlinessEvidence reports whether any behaviour signal has been verified.
// honorsGPC is deliberately excluded: it is a governance signal, not a
// cleanliness one, so a lone honorsGPC value (Yes or No) must not lift a site out
// of Unclassified — that is what keeps "No never beats Unknown" true even at the
// bottom of the scale, where an ad-inapplicable honorsGPC=No is inert.
func hasCleanlinessEvidence(s Signals) bool {
	return s.AdTrackingCookies != Unknown ||
		s.AdsTrackers != LevelUnknown ||
		s.ThirdPartyScripts != LevelUnknown ||
		s.Fingerprinting != Unknown ||
		s.SessionReplay != Unknown ||
		s.SellsSharesData != Unknown
}
