package classify

import (
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC) }

// TestDefaultDatasetLoads is the shipped-data guardrail: the embedded dataset
// must parse and validate, and the reference examples must grade as documented.
func TestDefaultDatasetLoads(t *testing.T) {
	c, err := New(WithClock(fixedNow))
	if err != nil {
		t.Fatalf("New with default dataset: %v", err)
	}

	yt := c.Classify("https://www.youtube.com/watch?v=abc")
	if yt.Grade != GradeF {
		t.Errorf("youtube.com grade = %v, want F", yt.Grade)
	}

	nc := c.Classify("https://www.youtube-nocookie.com/embed/abc")
	if nc.Grade != GradeC {
		t.Errorf("youtube-nocookie.com grade = %v, want C", nc.Grade)
	}
}

// TestValidateEntries enforces the honesty guardrails on hand-authored data.
func TestValidateEntries(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"missing verified on audited", `{"a.com":{"trust":"audited"}}`, true},
		{"invalid verified format", `{"a.com":{"trust":"audited","verified":"07/25/2026"}}`, true},
		{"future verified date", `{"a.com":{"trust":"audited","verified":"2099-01-01"}}`, true},
		{"own needs no verified", `{"a.com":{"trust":"own"}}`, false},
		{"valid audited entry", `{"a.com":{"trust":"audited","verified":"2026-01-01"}}`, false},
		{"invalid enum value", `{"a.com":{"trust":"audited","verified":"2026-01-01","signals":{"honorsGPC":"maybe"}}}`, true},
		{"unknown field typo", `{"a.com":{"trust":"audited","verified":"2026-01-01","bogus":true}}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(WithoutDefaultData(), WithDataBytes([]byte(tt.data)), WithClock(fixedNow))
			if (err != nil) != tt.wantErr {
				t.Errorf("New() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
