package classify

import "testing"

func TestGrade(t *testing.T) {
	tests := []struct {
		name  string
		trust Trust
		sig   Signals
		want  Grade
	}{
		{
			name:  "own site is always A",
			trust: TrustOwn,
			want:  GradeA,
		},
		{
			name: "fully verified clean site is A",
			sig: Signals{
				HonorsGPC: Yes, AdTrackingCookies: No,
				AdsTrackers: LevelNone, ThirdPartyScripts: LevelNone,
				Fingerprinting: No, SessionReplay: No, SellsSharesData: No,
			},
			want: GradeA,
		},
		{
			name: "clean but loads minor third-party is B",
			sig: Signals{
				HonorsGPC: Yes, AdTrackingCookies: No,
				AdsTrackers: LevelLow, ThirdPartyScripts: LevelLow,
				Fingerprinting: No, SessionReplay: No, SellsSharesData: No,
			},
			want: GradeB,
		},
		{
			name: "gpc and cookies confirmed but rest unknown is C",
			sig:  Signals{HonorsGPC: Yes, AdTrackingCookies: No},
			want: GradeC,
		},
		{
			// honorsGPC=No is inert when the site neither sells/shares nor
			// ad-tracks — nothing to opt out of. It must not cap at C, and (see
			// TestHonorsGPCNeverBeatsUnknown) must not beat leaving GPC Unknown.
			name: "gpc=no with no ad-tracking is inert (Unclassified)",
			sig:  Signals{HonorsGPC: No},
			want: GradeUnclassified,
		},
		{
			// GPC applicable (site ad-tracks): honorsGPC=No is a governance
			// failure -> C. This is the youtube-nocookie shape.
			name: "gpc=no on an ad-tracking site is a governance failure (C)",
			sig:  Signals{HonorsGPC: No, AdsTrackers: LevelLow, ThirdPartyScripts: LevelHigh},
			want: GradeC,
		},
		{
			// Same site with GPC Unknown also lands at C — recording No must not
			// improve the grade over Unknown (no inversion).
			name: "gpc=unknown on the same ad-tracking site is also C",
			sig:  Signals{AdsTrackers: LevelLow, ThirdPartyScripts: LevelHigh},
			want: GradeC,
		},
		{
			name: "ad cookies alone cap at D",
			sig:  Signals{AdTrackingCookies: Yes, HonorsGPC: Yes},
			want: GradeD,
		},
		{
			name: "session replay alone caps at D",
			sig:  Signals{SessionReplay: Yes},
			want: GradeD,
		},
		{
			name: "fingerprinting alone caps at D",
			sig:  Signals{Fingerprinting: Yes},
			want: GradeD,
		},
		{
			name: "cookies plus no gpc is F",
			sig:  Signals{AdTrackingCookies: Yes, HonorsGPC: No},
			want: GradeF,
		},
		{
			name: "fingerprinting plus heavy trackers is F",
			sig:  Signals{Fingerprinting: Yes, AdsTrackers: LevelHigh},
			want: GradeF,
		},
		{
			name: "no signals at all is Unclassified",
			sig:  Signals{},
			want: GradeUnclassified,
		},
		{
			// A confirmed-clean site with GPC Unknown and no sale/share must
			// reach B — it must NOT be stranded at "?" (the old positive gate).
			name: "confirmed clean, gpc unknown, no sale/share is B",
			sig: Signals{
				AdTrackingCookies: No, AdsTrackers: LevelNone, ThirdPartyScripts: LevelNone,
				Fingerprinting: No, SessionReplay: No, SellsSharesData: No,
			},
			want: GradeB,
		},
		{
			// A site whose only cookies are first-party (curated as No under the
			// tightened definition) plus minor content is B, not an automatic D.
			name: "first-party-only cookies with minor content is not a hard D",
			sig:  Signals{AdTrackingCookies: No, AdsTrackers: LevelLow, ThirdPartyScripts: LevelLow},
			want: GradeB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reasons := grade(tt.sig, tt.trust)
			if got != tt.want {
				t.Errorf("grade() = %v, want %v", got, tt.want)
			}
			if len(reasons) == 0 {
				t.Error("grade() returned no reasons; every badge must be explainable")
			}
		})
	}
}

// TestHonorsGPCNeverBeatsUnknown pins the two directional invariants for the GPC
// signal across a spread of base sites: honorsGPC=No must never grade better
// than Unknown, and honorsGPC=Yes must never grade worse (it is a booster). The
// Grade enum is ordinal (higher = better), so the check is numeric.
func TestHonorsGPCNeverBeatsUnknown(t *testing.T) {
	bases := map[string]Signals{
		"fully clean":      {AdTrackingCookies: No, AdsTrackers: LevelNone, ThirdPartyScripts: LevelNone, Fingerprinting: No, SessionReplay: No, SellsSharesData: No},
		"clean minor ads":  {AdTrackingCookies: No, AdsTrackers: LevelLow, ThirdPartyScripts: LevelLow, Fingerprinting: No, SessionReplay: No, SellsSharesData: No},
		"ad-tracking only": {AdsTrackers: LevelLow},
		"ad cookies":       {AdTrackingCookies: Yes},
		"heavy trackers":   {AdsTrackers: LevelHigh},
		"sells data":       {SellsSharesData: Yes},
		"no signals":       {},
	}
	for name, base := range bases {
		t.Run(name, func(t *testing.T) {
			withNo := base
			withNo.HonorsGPC = No
			withUnknown := base
			withUnknown.HonorsGPC = Unknown
			withYes := base
			withYes.HonorsGPC = Yes

			gNo, _ := grade(withNo, TrustUnknown)
			gUnk, _ := grade(withUnknown, TrustUnknown)
			gYes, _ := grade(withYes, TrustUnknown)

			if gNo > gUnk {
				t.Errorf("honorsGPC=No (%v) graded better than Unknown (%v)", gNo, gUnk)
			}
			if gYes < gUnk {
				t.Errorf("honorsGPC=Yes (%v) graded worse than Unknown (%v)", gYes, gUnk)
			}
		})
	}
}
