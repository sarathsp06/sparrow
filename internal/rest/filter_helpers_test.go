package rest

import (
	"testing"
	"time"
)

func TestParseLabelFilter(t *testing.T) {
	labels, err := parseLabelFilter("env=prod, region=us-east-1")
	if err != nil {
		t.Fatalf("parseLabelFilter returned error: %v", err)
	}
	if got, want := labels["env"], "prod"; got != want {
		t.Fatalf("env = %q, want %q", got, want)
	}
	if got, want := labels["region"], "us-east-1"; got != want {
		t.Fatalf("region = %q, want %q", got, want)
	}

	if _, err := parseLabelFilter("env"); err == nil {
		t.Fatal("expected invalid label filter to fail")
	}
}

func TestParseDateFilter(t *testing.T) {
	start, err := parseDateFilter("2026-09-18", false)
	if err != nil {
		t.Fatalf("parseDateFilter start returned error: %v", err)
	}
	if got, want := start.Format(time.RFC3339Nano), "2026-09-18T00:00:00Z"; got != want {
		t.Fatalf("start = %q, want %q", got, want)
	}

	end, err := parseDateFilter("2026-09-18", true)
	if err != nil {
		t.Fatalf("parseDateFilter end returned error: %v", err)
	}
	if got, want := end.Format(time.RFC3339Nano), "2026-09-18T23:59:59.999999999Z"; got != want {
		t.Fatalf("end = %q, want %q", got, want)
	}

	if _, err := parseDateFilter("09/18/2026", false); err == nil {
		t.Fatal("expected invalid date filter to fail")
	}
}
