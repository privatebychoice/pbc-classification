package main

import (
	"reflect"
	"testing"

	classify "go.privatebychoice.com/pbc-classification"
)

func TestToResult(t *testing.T) {
	in := classify.Classification{
		Input:    "https://youtube.com",
		Domain:   "youtube.com",
		Matched:  true,
		Grade:    classify.GradeF,
		Trust:    classify.TrustAudited,
		Verified: "2026-07-25",
		Reasons:  []string{"Sets ad/tracking cookies"},
	}
	got := toResult(in)
	want := result{
		URL:       "https://youtube.com",
		Domain:    "youtube.com",
		Matched:   true,
		Grade:     "F",
		GradeName: "Invasive",
		Trust:     "audited",
		Verified:  "2026-07-25",
		Reasons:   []string{"Sets ad/tracking cookies"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toResult() = %+v, want %+v", got, want)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a.com", []string{"a.com"}},
		{" a.com , b.com ,", []string{"a.com", "b.com"}},
	}
	for _, tt := range tests {
		if got := splitCSV(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
