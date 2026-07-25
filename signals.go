package classify

import (
	"fmt"
	"strings"
)

// Ternary is a three-valued privacy signal. The zero value is Unknown, so an
// absent or unverified field is honestly treated as "not confirmed" rather than
// as a false pass. This is the cornerstone of honest scoring: absence of
// evidence is never treated as evidence of privacy.
type Ternary uint8

const (
	// Unknown means the signal has not been verified. It must never raise a grade.
	Unknown Ternary = iota
	// No means the (undesirable) behaviour was verified to be absent.
	No
	// Yes means the (undesirable) behaviour was verified to be present.
	Yes
)

func (t Ternary) String() string {
	switch t {
	case No:
		return "no"
	case Yes:
		return "yes"
	default:
		return "unknown"
	}
}

// UnmarshalText accepts "yes"/"no"/"unknown" (plus "true"/"false"); an empty
// value is Unknown. Any other value is rejected so typos fail loudly at load.
func (t *Ternary) UnmarshalText(b []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(b))) {
	case "", "unknown":
		*t = Unknown
	case "no", "false":
		*t = No
	case "yes", "true":
		*t = Yes
	default:
		return fmt.Errorf("classify: invalid ternary %q (want yes|no|unknown)", string(b))
	}
	return nil
}

func (t Ternary) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

// Level is a four-valued magnitude for quantities like ad/tracker density or
// third-party script usage. The zero value is LevelUnknown.
type Level uint8

const (
	LevelUnknown Level = iota
	LevelNone
	LevelLow
	LevelHigh
)

func (l Level) String() string {
	switch l {
	case LevelNone:
		return "none"
	case LevelLow:
		return "low"
	case LevelHigh:
		return "high"
	default:
		return "unknown"
	}
}

// UnmarshalText accepts the canonical "none"/"low"/"high"/"unknown" plus the
// friendlier aliases used in prose ("some"/"few" -> low, "heavy"/"many" -> high).
func (l *Level) UnmarshalText(b []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(b))) {
	case "", "unknown":
		*l = LevelUnknown
	case "none":
		*l = LevelNone
	case "low", "some", "few":
		*l = LevelLow
	case "high", "heavy", "many":
		*l = LevelHigh
	default:
		return fmt.Errorf("classify: invalid level %q (want none|low|high|unknown)", string(b))
	}
	return nil
}

func (l Level) MarshalText() ([]byte, error) { return []byte(l.String()), nil }

// Signals captures what is known about a destination's privacy behaviour. Every
// field defaults to its Unknown value so a partially-filled entry is honest
// about what has not been verified.
type Signals struct {
	// AdTrackingCookies: sets advertising/tracking cookies (a grade disqualifier).
	AdTrackingCookies Ternary `json:"adTrackingCookies,omitempty"`
	// HonorsGPC: honours the Global Privacy Control signal.
	HonorsGPC Ternary `json:"honorsGPC,omitempty"`
	// AdsTrackers: density of third-party ads and trackers.
	AdsTrackers Level `json:"adsTrackers,omitempty"`
	// ThirdPartyScripts: volume of third-party JavaScript.
	ThirdPartyScripts Level `json:"thirdPartyScripts,omitempty"`
	// Fingerprinting: uses canvas/WebGL/audio device fingerprinting (a disqualifier).
	Fingerprinting Ternary `json:"fingerprinting,omitempty"`
	// SessionReplay: records sessions / keystrokes (Hotjar, FullStory, ...) (a disqualifier).
	SessionReplay Ternary `json:"sessionReplay,omitempty"`
	// SellsSharesData: sells or shares personal data, in the CCPA sense (a disqualifier).
	SellsSharesData Ternary `json:"sellsSharesData,omitempty"`
	// ThirdPartyDomains: count of distinct third-party domains contacted.
	// Informational only (feeds Reasons); nil means not measured.
	ThirdPartyDomains *int `json:"thirdPartyDomains,omitempty"`
}
