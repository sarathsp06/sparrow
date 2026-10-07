package store

import (
	"testing"
	"time"
)

func TestHealthLabel(t *testing.T) {
	cases := []struct {
		name                string
		events              int
		rate                float64
		consecutiveFailures int
		want                string
	}{
		{"no data", 0, 0, 0, "unknown"},
		{"too few events", 2, 1.0, 0, "unknown"},
		{"healthy", 3, 1.0, 0, "healthy"},
		{"degraded", 5, 0.85, 0, "degraded"},
		{"unhealthy by rate", 10, 0.5, 0, "unhealthy"},
		{"unhealthy by run", 3, 1.0, 5, "unhealthy"},
		{"low rate but few events", 4, 0.5, 0, "unknown"},
	}
	for _, c := range cases {
		if got := healthLabel(c.events, c.rate, c.consecutiveFailures); got != c.want {
			t.Errorf("%s: healthLabel(%d, %.2f, %d) = %q, want %q", c.name, c.events, c.rate, c.consecutiveFailures, got, c.want)
		}
	}
}

func TestHealthLabelDue(t *testing.T) {
	now := time.Now()
	recent := now.Add(-time.Second)
	stale := now.Add(-healthLabelRecomputeInterval)

	cases := []struct {
		name string
		due  bool
		got  bool
	}{
		{"first outcome ever", true, healthLabelDue(now, nil, 0, 0, false, false, true)},
		{"label never computed", true, healthLabelDue(now, nil, 0, 0, false, true, true)},
		{"steady successes, computed recently", false, healthLabelDue(now, &recent, 0, 0, false, true, true)},
		{"steady failures below threshold, computed recently", false, healthLabelDue(now, &recent, 2, 3, true, true, false)},
		{"interval elapsed", true, healthLabelDue(now, &stale, 0, 0, false, true, true)},
		{"success after failure", true, healthLabelDue(now, &recent, 2, 0, true, true, true)},
		{"failure after success", true, healthLabelDue(now, &recent, 0, 1, false, true, false)},
		{"crossing five failures", true, healthLabelDue(now, &recent, 4, 5, true, true, false)},
		{"already past five failures", false, healthLabelDue(now, &recent, 7, 8, true, true, false)},
	}
	for _, c := range cases {
		if c.got != c.due {
			t.Errorf("%s: due = %v, want %v", c.name, c.got, c.due)
		}
	}
}
