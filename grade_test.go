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
			name: "does not honor gpc caps at C",
			sig:  Signals{HonorsGPC: No},
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
			name: "nothing bad found but gpc unknown is Unclassified",
			sig:  Signals{AdTrackingCookies: No, AdsTrackers: LevelNone},
			want: GradeUnclassified,
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
