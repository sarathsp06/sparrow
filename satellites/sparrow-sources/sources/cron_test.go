package sources

import (
	"testing"
	"time"
)

func mustParse(t *testing.T, expr string) *cronSpec {
	t.Helper()
	spec, err := parseCron(expr)
	if err != nil {
		t.Fatalf("parseCron(%q): %v", expr, err)
	}
	return spec
}

func at(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseCronInvalid(t *testing.T) {
	for _, expr := range []string{
		"",            // empty
		"* * * *",     // 4 fields
		"* * * * * *", // 6 fields
		"60 * * * *",  // minute out of range
		"* 24 * * *",  // hour out of range
		"* * 0 * *",   // dom out of range
		"* * 32 * *",  // dom out of range
		"* * * 13 *",  // month out of range
		"* * * * 8",   // dow out of range
		"5-1 * * * *", // inverted range
		"*/0 * * * *", // zero step
		"*/x * * * *", // garbage step
		"a * * * *",   // garbage value
		"1-b * * * *", // garbage range end
	} {
		if _, err := parseCron(expr); err == nil {
			t.Errorf("parseCron(%q): expected error", expr)
		}
	}
}

func TestCronMatch(t *testing.T) {
	// 2026-09-11 is a Friday (dow 5).
	tests := []struct {
		expr string
		time string
		want bool
	}{
		// wildcard
		{"* * * * *", "2026-09-11 10:30", true},
		// exact values
		{"30 10 11 9 *", "2026-09-11 10:30", true},
		{"31 10 11 9 *", "2026-09-11 10:30", false},
		{"30 11 11 9 *", "2026-09-11 10:30", false},
		{"30 10 12 9 *", "2026-09-11 10:30", false},
		{"30 10 11 8 *", "2026-09-11 10:30", false},
		// steps
		{"*/5 * * * *", "2026-09-11 10:30", true},
		{"*/5 * * * *", "2026-09-11 10:31", false},
		{"*/15 * * * *", "2026-09-11 10:45", true},
		// ranges and lists
		{"0-40 * * * *", "2026-09-11 10:30", true},
		{"0-20 * * * *", "2026-09-11 10:30", false},
		{"10,20,30 * * * *", "2026-09-11 10:30", true},
		{"10,20,31 * * * *", "2026-09-11 10:30", false},
		// range with step
		{"0-59/10 * * * *", "2026-09-11 10:30", true},
		{"0-59/10 * * * *", "2026-09-11 10:35", false},
		// dow: Friday is 5; 7 folds to Sunday
		{"30 10 * * 5", "2026-09-11 10:30", true},
		{"30 10 * * 4", "2026-09-11 10:30", false},
		{"0 0 * * 7", "2026-09-13 00:00", true}, // Sunday as 7
		{"0 0 * * 0", "2026-09-13 00:00", true}, // Sunday as 0
		// dom/dow both restricted: OR semantics
		{"30 10 1 * 5", "2026-09-11 10:30", true},  // dow matches, dom doesn't
		{"30 10 11 * 2", "2026-09-11 10:30", true}, // dom matches, dow doesn't
		{"30 10 1 * 2", "2026-09-11 10:30", false}, // neither matches
		// only dom restricted: AND with wildcard dow
		{"30 10 11 * *", "2026-09-11 10:30", true},
		{"30 10 12 * *", "2026-09-11 10:30", false},
	}
	for _, tt := range tests {
		spec := mustParse(t, tt.expr)
		if got := spec.Match(at(tt.time)); got != tt.want {
			t.Errorf("%q.Match(%s) = %v, want %v", tt.expr, tt.time, got, tt.want)
		}
	}
}
