package classify

import (
	"testing"
	"time"
)

const fixture = `{
  "example.com": {"trust":"own","note":"mine"},
  "tracker.co.uk": {"trust":"audited","verified":"2026-01-01","signals":{"adTrackingCookies":"yes","honorsGPC":"no"}},
  "old.example.org": {"trust":"audited","verified":"2020-01-01","signals":{"honorsGPC":"yes","adTrackingCookies":"no","adsTrackers":"none","thirdPartyScripts":"none","fingerprinting":"no","sessionReplay":"no","sellsSharesData":"no"}}
}`

func newTestClassifier(t *testing.T, data string, now time.Time) *Classifier {
	t.Helper()
	c, err := New(
		WithoutDefaultData(),
		WithDataBytes([]byte(data)),
		WithClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestClassify(t *testing.T) {
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	c := newTestClassifier(t, fixture, now)

	tests := []struct {
		name      string
		url       string
		wantMatch bool
		wantGrade Grade
		wantTrust Trust
		wantStale bool
		wantDom   string
	}{
		{"own site matched via subdomain", "https://www.example.com/page", true, GradeA, TrustOwn, false, "example.com"},
		{"multi-part TLD resolves correctly", "https://sub.tracker.co.uk/x", true, GradeF, TrustAudited, false, "tracker.co.uk"},
		{"exact-host entry wins", "http://old.example.org", true, GradeUnclassified, TrustAudited, true, "old.example.org"},
		{"unknown domain is Unclassified", "https://nowhere.test/", false, GradeUnclassified, TrustUnknown, false, "nowhere.test"},
		{"bare host without scheme", "tracker.co.uk", true, GradeF, TrustAudited, false, "tracker.co.uk"},
		{"unparseable input", "http://[", false, GradeUnclassified, TrustUnknown, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Classify(tt.url)
			if got.Matched != tt.wantMatch {
				t.Errorf("Matched = %v, want %v", got.Matched, tt.wantMatch)
			}
			if got.Grade != tt.wantGrade {
				t.Errorf("Grade = %v, want %v", got.Grade, tt.wantGrade)
			}
			if got.Trust != tt.wantTrust {
				t.Errorf("Trust = %v, want %v", got.Trust, tt.wantTrust)
			}
			if got.Stale != tt.wantStale {
				t.Errorf("Stale = %v, want %v", got.Stale, tt.wantStale)
			}
			if got.Domain != tt.wantDom {
				t.Errorf("Domain = %q, want %q", got.Domain, tt.wantDom)
			}
		})
	}
}

func TestWithFirstParty(t *testing.T) {
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	c, err := New(
		WithoutDefaultData(),
		WithFirstParty("privatebychoice.example", "www.tul.example"),
		WithClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, u := range []string{"https://privatebychoice.example/x", "https://tul.example/y"} {
		got := c.Classify(u)
		if got.Grade != GradeA || got.Trust != TrustOwn {
			t.Errorf("Classify(%q) = %v/%v, want A/own", u, got.Grade, got.Trust)
		}
	}
}

func TestStaleDisabled(t *testing.T) {
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	c, err := New(
		WithoutDefaultData(),
		WithDataBytes([]byte(fixture)),
		WithStaleAfter(0), // disable staleness
		WithClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := c.Classify("http://old.example.org")
	if got.Stale {
		t.Error("expected staleness disabled")
	}
	if got.Grade != GradeA {
		t.Errorf("Grade = %v, want A (clean signals, staleness off)", got.Grade)
	}
}
